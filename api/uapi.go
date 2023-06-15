// Binds onto eureka uapi
package api

import (
	"errors"
	"net/http"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/exp/slices"
)

const (
	TargetTypeUser   = "user"
	TargetTypeBot    = "bot"
	TargetTypeServer = "server"
)

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
			var vanity pgtype.Text
			err := state.Pool.QueryRow(state.Context, "SELECT bot_id, vanity FROM bots WHERE api_token = $1", strings.Replace(authHeader, "Bot ", "", 1)).Scan(&id, &vanity)

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
			urlIds = []string{id.String, vanity.String}
		}

		// Now handle the URLVar
		if auth.URLVar != "" {
			state.Logger.Info("URLVar: ", auth.URLVar)
			gotUserId := chi.URLParam(req, auth.URLVar)
			if !slices.Contains(urlIds, gotUserId) {
				authData = uapi.AuthData{} // Remove auth data
			}
		}

		// Banned users cannot use the API at all otherwise if not explicitly scoped to "ban_exempt"
		if authData.Banned && auth.AllowedScope != "ban_exempt" {
			return uapi.AuthData{}, uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json: types.ApiError{
					Error:   true,
					Message: "You are banned from the list. If you think this is a mistake, please contact support.",
				},
			}, false
		}
	}

	if len(r.Auth) > 0 && !authData.Authorized && !r.AuthOptional {
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusUnauthorized), false
	}

	return authData, uapi.HttpResponse{}, true
}

func RouteDataMiddleware(r *uapi.RouteData, req *http.Request) (*uapi.RouteData, error) {
	clientHeader := req.Header.Get("X-Client")

	var isClient bool
	if clientHeader != "" {
		if clientHeader != state.Config.Meta.CliNonce {
			return nil, errors.New("out-of-date client")
		}

		isClient = true
	}

	r.Props = map[string]string{
		"isClient": func() string {
			if isClient {
				return "1"
			}
			return "0"
		}(),
	}

	return r, nil
}

func IsClient(r *http.Request) bool {
	clientHeader := r.Header.Get("X-Client")

	if clientHeader != "" {
		return clientHeader == state.Config.Meta.CliNonce
	}

	return false
}

// Only used during development, should only be used when rewriting
// endpoints itself
func ClientSupports(r *http.Request, feature string) bool {
	features := r.Header.Get("X-Client-Compat")

	if features == "" {
		return false
	}

	featuresArr := strings.Split(features, ",")

	return slices.Contains(featuresArr, feature)
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
		RouteDataMiddleware: RouteDataMiddleware,
		Redis:               state.Redis,
		Context:             state.Context,
	})
}
