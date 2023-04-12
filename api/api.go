// Defines a standard way to define routes
package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"popplio/constants"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/exp/slices"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Stores the current tag
var CurrentTag string

// A API Router, not to be confused with Router which routes the actual routes
type APIRouter interface {
	Routes(r *chi.Mux)
	Tag() (string, string)
}

type TargetType int

const (
	TargetTypeUser TargetType = iota
	TargetTypeBot
	TargetTypeServer
)

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
	URLVar       string
	Type         TargetType
	AllowedScope string // If this is set, then ban checks are not fatal
}

type AuthData struct {
	TargetType TargetType `json:"target_type"`
	ID         string     `json:"id"`
	Authorized bool       `json:"authorized"`
	Banned     bool       `json:"banned"` // Only applicable with AllowedScope
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
	Context  context.Context
	Auth     AuthData
	IsClient bool
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

		var urlIds []string

		switch auth.Type {
		case TargetTypeUser:
			// Check if the user exists with said API token only
			var id pgtype.Text
			var banned bool

			err := state.Pool.QueryRow(state.Context, "SELECT user_id, banned FROM users WHERE api_token = $1", strings.Replace(authHeader, "User ", "", 1)).Scan(&id, &banned)

			if err != nil {
				continue
			}

			if !id.Valid {
				continue
			}

			authData = AuthData{
				TargetType: TargetTypeUser,
				ID:         id.String,
				Authorized: true,
				Banned:     banned,
			}
			urlIds = []string{id.String}
		case TargetTypeBot:
			// Check if the bot exists with said token only
			var id pgtype.Text
			var vanity pgtype.Text
			err := state.Pool.QueryRow(state.Context, "SELECT bot_id, vanity FROM bots WHERE api_token = $1", strings.Replace(authHeader, "Bot ", "", 1)).Scan(&id, &vanity)

			if err != nil {
				continue
			}

			if !id.Valid {
				continue
			}

			authData = AuthData{
				TargetType: TargetTypeBot,
				ID:         id.String,
				Authorized: true,
			}
			urlIds = []string{id.String, vanity.String}
		}

		// Now handle the URLVar
		if auth.URLVar != "" {
			state.Logger.Info("URLVar: ", auth.URLVar)
			gotUserId := chi.URLParam(req, auth.URLVar)
			if !slices.Contains(urlIds, gotUserId) {
				authData = AuthData{} // Remove auth data
			}
		}

		// Banned users cannot use the API at all otherwise if not explicitly scoped to "ban_exempt"
		if authData.Banned && auth.AllowedScope != "ban_exempt" {
			return AuthData{}, HttpResponse{
				Status: http.StatusForbidden,
				Json: types.ApiError{
					Error:   true,
					Message: "You are banned from the list. If you think this is a mistake, please contact support.",
				},
			}, false
		}
	}

	if len(r.Auth) > 0 && !authData.Authorized && !r.AuthOptional {
		return AuthData{}, DefaultResponse(http.StatusUnauthorized), false
	}

	return authData, HttpResponse{}, true
}

type TestData struct {
	Route  func(d RouteData, r *http.Request) HttpResponse
	Body   []byte
	T      *testing.T
	Params map[string]string
	AuthID string
}

func Test(d TestData) {
	// Open config.yaml
	os.Chdir("../../../../")

	// Create new ctx
	rctx := context.Background()

	ctx := chi.NewRouteContext()

	for k, v := range d.Params {
		ctx.URLParams.Add(k, v)
	}

	rctx = context.WithValue(rctx, chi.RouteCtxKey, ctx)

	state.Setup()

	id := os.Getenv("TEST__USER_ID")

	if d.AuthID != "" {
		id = d.AuthID
	}

	testRouteData := RouteData{
		Context:  rctx,
		IsClient: true,
		Auth: AuthData{
			ID:         id,
			Authorized: true,
		},
	}

	// Create a test request
	req := http.Request{
		Body: io.NopCloser(bytes.NewReader(d.Body)),
	}

	ctxReq := req.WithContext(rctx)

	resp := d.Route(testRouteData, ctxReq)

	if resp.Status != 0 && resp.Status != http.StatusOK && resp.Status != http.StatusCreated {
		d.T.Error("Expected status 200 or 204 but got ", strconv.Itoa(resp.Status), resp)
		return
	}

	d.T.Log(resp)
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

	docsObj := r.Docs()

	docsObj.Pattern = r.Pattern
	docsObj.OpId = r.OpId
	docsObj.Method = r.Method.String()
	docsObj.Tags = []string{CurrentTag}
	docsObj.AuthType = []string{}

	for _, auth := range r.Auth {
		switch auth.Type {
		case TargetTypeUser:
			docsObj.AuthType = append(docsObj.AuthType, "User")
		case TargetTypeBot:
			docsObj.AuthType = append(docsObj.AuthType, "Bot")
		default:
			panic("Unknown auth type: " + r.String())
		}
	}

	// Count the number of { and } in the pattern
	brStart := strings.Count(r.Pattern, "{")
	brEnd := strings.Count(r.Pattern, "}")
	pathParams := []string{}
	patternParams := []string{}

	for _, param := range docsObj.Params {
		if param.In == "" || param.Name == "" || param.Schema == nil {
			panic("Param is missing required fields: " + r.String())
		}

		if param.In == "path" {
			pathParams = append(pathParams, param.Name)
		}
	}

	// Get pattern params from the pattern
	for _, param := range strings.Split(r.Pattern, "/") {
		if strings.HasPrefix(param, "{") && strings.HasSuffix(param, "}") {
			patternParams = append(patternParams, param[1:len(param)-1])
		} else if strings.Contains(param, "{") || strings.Contains(param, "}") {
			panic("{ and } in pattern but does not start with it " + r.String())
		}
	}

	if brStart != brEnd {
		panic("Mismatched { and } in pattern: " + r.String())
	}

	if brStart != len(pathParams) {
		panic("Mismatched number of params and { in pattern: " + r.String())
	}

	if !slices.Equal(patternParams, pathParams) {
		panic("Mismatched params in pattern and docs: " + r.String())
	}

	// Add the path params to the docs
	docs.Route(docsObj)

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

			clientHeader := req.Header.Get("X-Client")

			var isClient bool
			if clientHeader != "" {
				if clientHeader != state.Config.Meta.CliNonce {
					resp <- HttpResponse{
						Status: http.StatusUnprocessableEntity,
						Json: types.ApiError{
							Error:   true,
							Message: "Out-of-date client",
						},
					}
					return
				}

				isClient = true
			}

			authData, httpResp, ok := r.Authorize(req)

			if !ok {
				resp <- httpResp
				return
			}

			resp <- r.Handler(RouteData{
				Context:  ctx,
				Auth:     authData,
				IsClient: isClient,
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
// 200 is treated as 204 No Content
func DefaultResponse(statusCode int) HttpResponse {
	switch statusCode {
	case http.StatusForbidden:
		return HttpResponse{
			Status: statusCode,
			Data:   constants.Forbidden,
		}
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
	case http.StatusNoContent, http.StatusOK:
		return HttpResponse{
			Status: http.StatusNoContent,
		}
	}

	return HttpResponse{
		Status: statusCode,
		Data:   constants.InternalError,
	}
}

// Read body
func marshalReq(r *http.Request, dst interface{}) (resp HttpResponse, ok bool) {
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
		}, false
	}

	return HttpResponse{}, true
}

func MarshalReq(r *http.Request, dst interface{}) (resp HttpResponse, ok bool) {
	return marshalReq(r, dst)
}

func MarshalReqWithHeaders(r *http.Request, dst interface{}, headers map[string]string) (resp HttpResponse, ok bool) {
	resp, err := marshalReq(r, dst)

	resp.Headers = headers

	return resp, err
}
