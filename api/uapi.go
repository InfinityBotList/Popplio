// Binds onto eureka uapi
package api

import (
	"net/http"
	"popplio/constants"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
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

// Authorizes a request
func Authorize(r uapi.Route, req *http.Request) (uapi.AuthData, uapi.HttpResponse, bool) {
	authHeader := req.Header.Get("Authorization")

	if len(r.Auth) > 0 && authHeader == "" && !r.AuthOptional {
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusUnauthorized), false
	}

	authData := uapi.AuthData{}

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

			authData = uapi.AuthData{
				TargetType: TargetTypeUser,
				ID:         id.String,
				Authorized: true,
				Banned:     banned,
			}
			urlIds = []string{id.String}
		case TargetTypeBot:
			// Check if the bot exists with said token only
			var id pgtype.Text
			err := state.Pool.QueryRow(state.Context, "SELECT bot_id FROM bots WHERE api_token = $1", strings.Replace(authHeader, "Bot ", "", 1)).Scan(&id)

			if err != nil {
				continue
			}

			if !id.Valid {
				continue
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeBot,
				ID:         id.String,
				Authorized: true,
			}
			urlIds = []string{id.String}
		case TargetTypeServer:
			// Check if the server exists with said token only
			var id pgtype.Text
			err := state.Pool.QueryRow(state.Context, "SELECT server_id FROM servers WHERE api_token = $1", strings.Replace(authHeader, "Server ", "", 1)).Scan(&id)

			if err != nil {
				continue
			}

			if !id.Valid {
				continue
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeServer,
				ID:         id.String,
				Authorized: true,
			}
			urlIds = []string{id.String}
		}

		// Now handle the URLVar
		if auth.URLVar != "" {
			state.Logger.Info("Checking URL variable against user ID from auth token", zap.String("URLVar", auth.URLVar))
			gotUserId := chi.URLParam(req, auth.URLVar)
			if !slices.Contains(urlIds, gotUserId) {
				authData = uapi.AuthData{} // Remove auth data
			}
		}

		// Banned users cannot use the API at all otherwise if not explicitly scoped to "ban_exempt"
		if authData.Banned && auth.AllowedScope != "ban_exempt" {
			return uapi.AuthData{}, uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You are banned from the list. If you think this is a mistake, please contact support."},
			}, false
		}
	}

	if len(r.Auth) > 0 && !authData.Authorized && !r.AuthOptional {
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusUnauthorized), false
	}

	return authData, uapi.HttpResponse{}, true
}

func Setup() {
	uapi.SetupState(uapi.UAPIState{
		Logger:    state.Logger,
		Authorize: Authorize,
		AuthTypeMap: map[string]string{
			TargetTypeUser:   "user",
			TargetTypeBot:    "bot",
			TargetTypeServer: "server",
		},
		Redis:   state.Redis,
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
