// Defines a standard way to define routes
package api

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/infinitybotlist/popplio/constants"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/state"
	"github.com/infinitybotlist/popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Simple blank map to avoid creating maps for every marshal etc.
var BlankMap = make(map[string]string)

// Stores the current tag
var CurrentTag string

// A API Router, not to be confused with Router which routes the actual routes
type APIRouter interface {
	Routes(r *chi.Mux)
	Tag() (string, string)
}

type Method int

const (
	GET Method = iota
	POST
	PATCH
	PUT
	DELETE
	HEAD
)

// Returns the method as a string
func (m Method) String() string {
	switch m {
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PATCH:
		return "PATCH"
	case PUT:
		return "PUT"
	case DELETE:
		return "DELETE"
	case HEAD:
		return "HEAD"
	}

	panic("Invalid method")
}

type AuthType struct {
	URLVar string
	Type   types.TargetType
}

type AuthData struct {
	TargetType types.TargetType `json:"target_type"`
	ID         string           `json:"id"`
	Authorized bool             `json:"authorized"`
}

// Represents a route on the API
type Route struct {
	Method       Method
	Pattern      string
	OpId         string
	Handler      func(d RouteData, r *http.Request) HttpResponse
	Setup        func()
	Docs         func() *docs.Doc
	Auth         []AuthType
	AuthOptional bool
}

type RouteData struct {
	Context context.Context
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

func (r Route) String() string {
	return r.Method.String() + " " + r.Pattern + " (" + r.OpId + ")"
}

// Authorizes a request
func (r Route) Authorize(req *http.Request) (AuthData, HttpResponse, bool) {
	authHeader := req.Header.Get("Authorization")

	if len(r.Auth) > 0 && authHeader == "" && !r.AuthOptional {
		return AuthData{}, DefaultResponse(http.StatusUnauthorized), false
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
			return AuthData{}, HttpResponse{
				Status: http.StatusNotImplemented,
				Data:   "Server auth is not implemented yet",
			}, false
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
				err := state.Pool.QueryRow(state.Context, "SELECT bot_id FROM bots WHERE api_token = $1", strings.Replace(authHeader, "Bot ", "", 1)).Scan(&id)

				if err != nil {
					continue
				}

				if !id.Valid {
					continue
				}

				authData = AuthData{
					TargetType: types.TargetTypeBot,
					ID:         id.String,
					Authorized: true,
				}
			}
		}
	}

	if len(r.Auth) > 0 && !authData.Authorized && !r.AuthOptional {
		return AuthData{}, DefaultResponse(http.StatusUnauthorized), false
	}

	return authData, HttpResponse{}, true
}

func (r Route) Route(ro Router) {
	if r.OpId == "" {
		panic("OpId is empty: " + r.String())
	}

	if r.Handler == nil {
		panic("Handler is nil: " + r.String())
	}

	if r.Docs == nil {
		panic("Docs is nil: " + r.String())
	}

	if r.Pattern == "" {
		panic("Pattern is empty: " + r.String())
	}

	if CurrentTag == "" {
		panic("CurrentTag is empty: " + r.String())
	}

	if r.Setup != nil {
		r.Setup()
	}

	docs := r.Docs()

	if docs.OpId != r.OpId {
		panic("OpId requested by router does not match docs OpId: " + r.String())
	}

	if !docs.Added() {
		panic("added not set to true, docs.Route not called: " + r.String())
	}

	if docs.OpId == "" {
		panic("OpId is empty. Did you forget to set it: " + r.String())
	}

	if docs.Method == "" {
		panic("Method is empty:" + r.String())
	}

	// Ensure auth types matches auth types given
	if len(r.Auth) != len(docs.AuthType) {
		panic("Auth types does not match docs auth types: " + r.String())
	}

	// Ensure method matches method given
	if r.Method.String() != docs.Method {
		panic("Method does not match docs method: " + r.String())
	}

	for i, auth := range r.Auth {
		if auth.Type != docs.AuthType[i] {
			panic("Auth types does not match docs auth types (mismatched type): " + r.String())
		}
	}

	handle := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		resp := make(chan HttpResponse)

		go func() {
			defer func() {
				err := recover()

				if err != nil {
					state.Logger.Error(err)
					resp <- HttpResponse{
						Status: http.StatusInternalServerError,
						Data:   constants.InternalError,
					}
				}
			}()

			authData, httpResp, ok := r.Authorize(req)

			if !ok {
				resp <- httpResp
				return
			}

			resp <- r.Handler(RouteData{
				Context: ctx,
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
		panic("Unknown method for route: " + r.String())
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
			errorMsg = field + " [" + err.Tag() + "]"
		} else {
			errorMsg = err.Error()
		}

		if i == 0 {
			firstError = errorMsg
		}

		errors[err.StructField()] = errorMsg
	}

	return HttpResponse{
		Status: http.StatusBadRequest,
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

// Read body
func marshalReq(r *http.Request, dst interface{}, headers map[string]string) (resp HttpResponse, ok bool) {
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		return DefaultResponse(http.StatusInternalServerError), false
	}

	if len(bodyBytes) == 0 {
		return HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "A body is required for this endpoint",
				Error:   true,
			},
			Headers: headers,
		}, false
	}

	err = json.Unmarshal(bodyBytes, &dst)

	if err != nil {
		state.Logger.Error(err)
		return HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid JSON: " + err.Error(),
				Error:   true,
			},
			Headers: headers,
		}, false
	}

	return HttpResponse{}, true
}

func MarshalReq(r *http.Request, dst interface{}) (resp HttpResponse, ok bool) {
	return marshalReq(r, dst, BlankMap)
}

func MarshalReqWithHeaders(r *http.Request, dst interface{}, headers map[string]string) (resp HttpResponse, ok bool) {
	return marshalReq(r, dst, headers)
}
