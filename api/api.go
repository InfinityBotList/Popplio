// Defines a standard way to define routes
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"reflect"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Stores the current tag
var CurrentTag string

type Method = int

const (
	GET Method = iota
	POST
	PATCH
	PUT
	DELETE
	HEAD
)

type AuthType struct {
	URLVar string
	Type   types.TargetType
}

type AuthData struct {
	TargetType types.TargetType
	ID         string
	Authorized bool
}

type Route struct {
	Method       Method
	Pattern      string
	Handler      func(d RouteData, r *http.Request)
	Setup        func()
	Docs         func() *docs.Doc
	Auth         []AuthType
	AuthOptional bool
}

type RouteData struct {
	Context context.Context
	Resp    chan HttpResponse
	Auth    AuthData
}

type Router interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
}

func (r Route) Route(ro Router) {
	if r.Handler == nil {
		panic("Handler is nil")
	}

	if r.Docs == nil {
		panic("Docs is nil")
	}

	if r.Pattern == "" {
		panic("Pattern is empty")
	}

	if CurrentTag == "" {
		panic("CurrentTag is empty")
	}

	if r.Setup != nil {
		r.Setup()
	}

	docs := r.Docs()

	// Ensure auth types matches auth types given
	if len(r.Auth) != len(docs.AuthType) {
		panic("Auth types does not match docs auth types: " + r.Pattern)
	}

	for i, auth := range r.Auth {
		if auth.Type != docs.AuthType[i] {
			panic("Auth types does not match docs auth types (mismatched type): " + r.Pattern)
		}
	}

	handle := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		resp := make(chan HttpResponse)

		go func() {
			// Handle auth checks here
			authHeader := req.Header.Get("Authorization")

			if len(r.Auth) > 0 && authHeader == "" && !r.AuthOptional {
				resp <- DefaultResponse(http.StatusUnauthorized)
			}

			authData := AuthData{}

			for _, auth := range r.Auth {
				// There are two cases, one with a URLVar (such as /bots/stats) and one without

				if authData.Authorized {
					break
				}

				if authHeader == "" {
					continue
				}

				if auth.Type == types.TargetTypeServer {
					// Server auth
					resp <- HttpResponse{
						Status: http.StatusNotImplemented,
						Data:   "Server auth is not implemented yet",
					}
					return
				}

				if auth.URLVar != "" {
					targetId := chi.URLParam(req, auth.URLVar)

					if targetId == "" {
						state.Logger.With(
							zap.String("endpoint", r.Pattern),
							zap.String("urlVar", auth.URLVar),
						).Error("Target ID is empty")
						continue
					}

					switch auth.Type {
					case types.TargetTypeUser:
						// Check if the user exists with said ID and API token
						var id pgtype.Text
						err := state.Pool.QueryRow(state.Context, "SELECT user_id FROM users WHERE user_id = $1 AND api_token = $2", targetId, strings.Replace(authHeader, "User ", "", 1)).Scan(&id)

						if err != nil {
							continue
						}

						if !id.Valid || id.String != targetId {
							continue
						}

						authData = AuthData{
							TargetType: types.TargetTypeUser,
							ID:         targetId,
							Authorized: true,
						}
					case types.TargetTypeBot:
						// Check if the bot exists with said ID and API token
						var id pgtype.Text
						err := state.Pool.QueryRow(state.Context, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1) AND api_token = $2", targetId, strings.Replace(targetId, "Bot ", "", 1)).Scan(&id)

						if err != nil {
							continue
						}

						if !id.Valid || id.String != targetId {
							continue
						}

						authData = AuthData{
							TargetType: types.TargetTypeBot,
							ID:         targetId,
							Authorized: true,
						}
					}
				} else {
					// Case #2: No URLVar, only token
					switch auth.Type {
					case types.TargetTypeUser:
						// Check if the user exists with said API token only
						var id pgtype.Text
						err := state.Pool.QueryRow(state.Context, "SELECT user_id FROM users WHERE api_token = $1", strings.Replace(authHeader, "User ", "", 1)).Scan(&id)

						if err != nil {
							continue
						}

						if !id.Valid {
							continue
						}

						authData = AuthData{
							TargetType: types.TargetTypeUser,
							ID:         id.String,
							Authorized: true,
						}
					case types.TargetTypeBot:
						// Check if the bot exists with said token only
						var id pgtype.Text
						err := state.Pool.QueryRow(state.Context, "SELECT bot_id FROM bots WHERE token = $1", strings.Replace(authHeader, "Bot ", "", 1)).Scan(&id)

						if err != nil {
							continue
						}

						if !id.Valid {
							continue
						}
					}
				}
			}

			if len(r.Auth) > 0 && !authData.Authorized && !r.AuthOptional {
				resp <- DefaultResponse(http.StatusUnauthorized)
				return
			}

			r.Handler(RouteData{
				Context: ctx,
				Resp:    resp,
				Auth:    authData,
			}, req)
		}()

		respond(ctx, w, resp)
	}

	switch r.Method {
	case GET:
		ro.Get(r.Pattern, handle)
	case POST:
		ro.Post(r.Pattern, handle)
	case PATCH:
		ro.Patch(r.Pattern, handle)
	case PUT:
		ro.Put(r.Pattern, handle)
	case DELETE:
		ro.Delete(r.Pattern, handle)
	case HEAD:
		ro.Head(r.Pattern, handle)
	default:
		panic("Unknown method...")
	}
}

func respond(ctx context.Context, w http.ResponseWriter, data chan HttpResponse) {
	select {
	case <-ctx.Done():
		return
	case msg, ok := <-data:
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(constants.InternalError))
		}

		if msg.Redirect != "" {
			msg.Headers = map[string]string{
				"Location":     msg.Redirect,
				"Content-Type": "text/html; charset=utf-8",
			}
			msg.Data = "<a href=\"" + msg.Redirect + "\">Found</a>.\n"
			msg.Status = http.StatusFound
		}

		if len(msg.Headers) > 0 {
			for k, v := range msg.Headers {
				w.Header().Set(k, v)
			}
		}

		if msg.Json != nil {
			bytes, err := json.Marshal(msg.Json)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(constants.InternalError))
				return
			}

			// JSON needs this explicitly to avoid calling WriteHeader twice
			if msg.Status == 0 {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(msg.Status)
			}

			w.Write(bytes)

			if msg.CacheKey != "" && msg.CacheTime.Seconds() > 0 {
				go func() {
					err := state.Redis.Set(state.Context, msg.CacheKey, bytes, msg.CacheTime).Err()

					if err != nil {
						state.Logger.Error(err)
					}
				}()
			}
		}

		if msg.Status == 0 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(msg.Status)
		}

		if len(msg.Bytes) > 0 {
			w.Write(msg.Bytes)
		}

		w.Write([]byte(msg.Data))
		return
	}
}

type HttpResponse struct {
	// Data is the data to be sent to the client
	Data string
	// Optional, can be used in place of Data
	Bytes []byte
	// Json body to be sent to the client
	Json any
	// Headers to set
	Headers map[string]string
	// Status is the HTTP status code to send
	Status int
	// Cache the JSON to redis
	CacheKey string
	// Duration to cache the JSON for
	CacheTime time.Duration
	// Redirect to a URL
	Redirect string
}

func CompileValidationErrors(payload any) map[string]string {
	var errors = make(map[string]string)

	structType := reflect.TypeOf(payload)

	for _, f := range reflect.VisibleFields(structType) {
		errors[f.Name] = f.Tag.Get("msg")

		arrayMsg := f.Tag.Get("amsg")

		if arrayMsg != "" {
			errors[f.Name+"$arr"] = arrayMsg
		}
	}

	return errors
}

func ValidatorErrorResponse(compiled map[string]string, v validator.ValidationErrors) HttpResponse {
	var errors = make(map[string]string)

	firstError := ""

	for i, err := range v {
		fname := err.StructField()
		if strings.Contains(err.Field(), "[") {
			// We have a array response, so we need to get the array name
			fname = strings.Split(err.Field(), "[")[0] + "$arr"
		}

		field := compiled[fname]

		var errorMsg string
		if field != "" {
			errorMsg = field
		} else {
			errorMsg = err.Error()
		}

		if i == 0 {
			firstError = errorMsg
		}

		errors[err.StructField()] = errorMsg
	}

	return HttpResponse{
		Json: types.ApiError{
			Context: errors,
			Error:   true,
			Message: firstError,
		},
	}
}

// Creates a default HTTP response based on the status code
func DefaultResponse(statusCode int) HttpResponse {
	switch statusCode {
	case http.StatusUnauthorized:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.Unauthorized,
		}
	case http.StatusNotFound:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.NotFound,
		}
	case http.StatusBadRequest:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.BadRequest,
		}
	case http.StatusInternalServerError:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.InternalError,
		}
	case http.StatusMethodNotAllowed:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.MethodNotAllowed,
		}
	case http.StatusOK:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.Success,
		}
	}

	return HttpResponse{
		Status: statusCode,
		Data:   constants.InternalError,
	}
}
