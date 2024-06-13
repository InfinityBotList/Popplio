// Binds onto eureka uapi
package api

import (
	"errors"
	"net/http"
	"popplio/constants"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const (
	SESSION_EXPIRY = 60 * 30 // 30 minutes
)

const (
	TargetTypeUser   = "user"
	TargetTypeBot    = "bot"
	TargetTypeServer = "server"
)

type DefaultResponder struct{}

func (d DefaultResponder) New(err string, ctx map[string]string) any {
	return types.ApiError{
		Message: err,
		Context: ctx,
	}
}

// Returns the permission limits of a user
//
// Note: this is not actually used yet.
func PermLimits(d uapi.AuthData) []string {
	if !d.Authorized {
		return []string{}
	}

	permLimits, ok := d.Data["perm_limits"].([]string)

	if !ok {
		return []string{}
	}

	return permLimits
}

// Authorizes a request
func Authorize(r uapi.Route, req *http.Request) (uapi.AuthData, uapi.HttpResponse, bool) {
	if len(r.Auth) == 0 {
		return uapi.AuthData{}, uapi.HttpResponse{}, true
	}

	authHeader := req.Header.Get("Authorization")

	// If there is no auth header, and auth is not optional, return unauthorized
	//
	// Note that we do not set X-Session-Invalid here because the session is not invalid, it just has not been sent (likely due to a client bug?)
	if len(r.Auth) > 0 && authHeader == "" && !r.AuthOptional {
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusUnauthorized), false
	}

	authData := uapi.AuthData{}

	// Before doing anything else, delete expired/old auths first
	_, err := state.Pool.Exec(state.Context, "DELETE FROM api_sessions WHERE expiry < NOW()")

	if err != nil {
		state.Logger.Error("Failed to delete expired web API tokens [db delete]", zap.Error(err))
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Failed to delete expired web API tokens: " + err.Error()},
		}, false
	}

	// Get the prefix from the auth header, if any, by splitNing it into 2 parts
	// The first part is the prefix, the second part is the token (if len == 2)
	// Otherwise, prefix is empty
	var authPrefix string
	parts := strings.SplitN(authHeader, " ", 2)

	if len(parts) == 2 {
		authPrefix = strings.ToLower(parts[0])
		authHeader = parts[1]
	}

	// Check if the anything at all exists with said API token
	var sessId string
	var targetType string
	var targetId string
	var permLimits []string

	err = state.Pool.QueryRow(state.Context, "SELECT id, target_id, target_type, perm_limits FROM api_sessions WHERE token = $1", authHeader).Scan(&sessId, &targetId, &targetType, &permLimits)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid session token"},
			Headers: map[string]string{
				"X-Session-Invalid": "true",
			},
		}, false
	}

	if err != nil {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Could not fetch any sessions: " + err.Error()},
		}, false
	}

	if authPrefix != "" && authPrefix != targetType {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid authorization prefix, expected " + authPrefix + " but got " + targetType},
			Headers: map[string]string{
				"X-Session-Invalid": "true",
			},
		}, false
	}

	for _, auth := range r.Auth {
		// There are two cases, one with a URLVar (such as /bots/stats) and one without

		if authData.Authorized {
			break
		}

		if targetType != auth.Type {
			continue
		}

		switch auth.Type {
		case TargetTypeUser:
			var banned bool

			err := state.Pool.QueryRow(state.Context, "SELECT banned FROM users WHERE user_id = $1", targetId).Scan(&banned)

			if err != nil {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusInternalServerError,
					Json:   types.ApiError{Message: "Could not fetch user associated with this session: " + err.Error()},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeUser,
				ID:         targetId,
				Authorized: true,
				Banned:     banned,
			}
		case TargetTypeBot:
			var count int64
			err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", targetId).Scan(&count)

			if err != nil {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusInternalServerError,
					Json:   types.ApiError{Message: "Could not fetch count of bots associated with this session: " + err.Error()},
				}, false
			}

			if count == 0 {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusNotFound,
					Json:   types.ApiError{Message: "The bot associated with this session could not be found?"},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeBot,
				ID:         targetId,
				Authorized: true,
			}
		case TargetTypeServer:
			var count int64
			err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", targetId).Scan(&count)

			if err != nil {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusInternalServerError,
					Json:   types.ApiError{Message: "Could not fetch count of servers associated with this session: " + err.Error()},
				}, false
			}

			if count == 0 {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusNotFound,
					Json:   types.ApiError{Message: "The server associated with this session could not be found?"},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeServer,
				ID:         targetId,
				Authorized: true,
			}
		}

		// Now handle the URLVar
		if auth.URLVar != "" {
			state.Logger.Info("Checking URL variable against user ID from auth token", zap.String("URLVar", auth.URLVar))
			gotUserId := chi.URLParam(req, auth.URLVar)
			if gotUserId != targetId {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "You are not authorized to perform this action (URLVar does not match auth token)"},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}
		}

		// Banned users cannot use the API at all otherwise if not explicitly scoped to "ban_exempt"
		if authData.Banned && auth.AllowedScope != "ban_exempt" {
			return uapi.AuthData{}, uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You are banned from the list. If you think this is a mistake, please contact support."},
				Headers: map[string]string{
					"X-Session-Invalid": "true",
				},
			}, false
		}
	}

	authData.Data = map[string]any{
		"session_id":  sessId,
		"perm_limits": permLimits,
	}

	if !authData.Authorized && !r.AuthOptional {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "You are not authorized to perform this action"},
			Headers: map[string]string{
				"X-Session-Invalid": "true",
			},
		}, false
	}

	return authData, uapi.HttpResponse{}, true
}

func Setup() {
	uapi.SetupState(uapi.UAPIState{
		Logger:    state.Logger,
		Authorize: Authorize,
		AuthTypeMap: map[string]string{
			TargetTypeUser:   TargetTypeUser,
			TargetTypeBot:    TargetTypeBot,
			TargetTypeServer: TargetTypeServer,
		},
		Context: state.Context,
		Constants: &uapi.UAPIConstants{
			ResourceNotFound:    constants.ResourceNotFound,
			BadRequest:          constants.BadRequest,
			Forbidden:           constants.Forbidden,
			Unauthorized:        constants.Unauthorized,
			InternalServerError: constants.InternalServerError,
			MethodNotAllowed:    constants.MethodNotAllowed,
			BodyRequired:        constants.BodyRequired,
		},
		DefaultResponder: DefaultResponder{},
	})
}
