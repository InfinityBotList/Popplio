package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgtype"
	"golang.org/x/exp/slices"
)

const tagName = "Login"

var (
	clientInfo string // Filled in in routes

	allowedRedirectURLs = []string{
		"http://localhost:3000/api/login",               // DEV
		"https://reedwhisker.infinitybots.gg/api/login", // PROD
	}
)

type OauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
}

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to authorization and login (if this ever becomes publicly usable)"
}

func (b Router) Routes(r *chi.Mux) {
	clientInfo = `{"client_id":"` + os.Getenv("CLIENT_ID") + `"}`

	r.Route("/authorize", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/authorize/info",
			OpId:        "get_authorize_info",
			Summary:     "Get Login Info",
			Description: "Gets the login info such as the client ID to use for the login.",
			Tags:        []string{tagName},
			Resp:        types.AuthInfo{},
		})
		r.Get("/info", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(clientInfo))
		})

		docs.Route(&docs.Doc{
			Method: "GET",
			Path:   "/authorize",
			Params: []docs.Parameter{
				{
					Name:        "code",
					Description: "The code from the OAuth2 redirect",
					Required:    true,
					In:          "query",
					Schema:      docs.IdSchema,
				},
				{
					Name:        "redirect_uri",
					Description: "The redirect URI you used in the OAuth2 redirect",
					Required:    true,
					In:          "query",
					Schema:      docs.IdSchema,
				},
			},
			OpId:        "authorize",
			Summary:     "Login User",
			Description: "Takes in a ``code`` query parameter and returns a user ``token``. **Cannot be used outside of the site for security reasons but documented in case we wish to allow its use in the future.**",
			Tags:        []string{tagName},
			Resp:        types.AuthUser{},
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			redirectUri := r.URL.Query().Get("redirect_uri")
			if !slices.Contains(allowedRedirectURLs, redirectUri) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":true,"message":"Malformed redirect_uri"}`))
				return
			}

			if redirectUri == allowedRedirectURLs[0] {
				if r.Header.Get("Wistala-Server") != os.Getenv("DEV_WISTALA_SERVER_SECRET") {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error":true,"message":"This endpoint is not meant to be used by you"}`))
					return
				}
			} else {
				if r.Header.Get("Wistala-Server") != os.Getenv("WISTALA_SERVER_SECRET") {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error":true,"message":"This endpoint is not meant to be used by you"}`))
					return
				}
			}

			code := r.URL.Query().Get("code")

			if code == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":true,"message":"No code provided"}`))
				return
			}

			if state.Redis.Exists(state.Context, "codecache:"+code).Val() == 1 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":true,"message":"Code has been used before and is as such invalid"}`))
				return
			}

			state.Redis.Set(state.Context, "codecache:"+code, "0", 5*time.Minute)

			resp, err := http.PostForm("https://discord.com/api/v10/oauth2/token", url.Values{
				"client_id":     {os.Getenv("CLIENT_ID")},
				"client_secret": {os.Getenv("CLIENT_SECRET")},
				"grant_type":    {"authorization_code"},
				"code":          {code},
				"redirect_uri":  {redirectUri},
				"scope":         {"identify"},
			})

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to get token from Discord"}`))
				return
			}

			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to read response body"}`))
				return
			}

			var token struct {
				AccessToken string `json:"access_token"`
			}

			err = json.Unmarshal(body, &token)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to unmarshal response body from Discord"}`))
				return
			}

			if token.AccessToken == "" {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":true,"message":"No access token provided by Discord?"}`))
				return
			}

			cli := &http.Client{}

			req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to create request to Discord"}`))
				return
			}

			req.Header.Set("Authorization", "Bearer "+token.AccessToken)

			resp, err = cli.Do(req)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to get user from Discord"}`))
				return
			}

			defer resp.Body.Close()

			body, err = io.ReadAll(resp.Body)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to read response body"}`))
				return
			}

			var user OauthUser

			err = json.Unmarshal(body, &user)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to unmarshal response body from Discord"}`))
				return
			}

			if user.ID == "" {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":true,"message":"No user ID provided by Discord?"}`))
				return
			}

			// Check if user exists on database
			var exists bool

			err = state.Pool.QueryRow(state.Context, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", user.ID).Scan(&exists)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to check if user exists"}`))
				return
			}

			discordUser, err := utils.GetDiscordUser(user.ID)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to get user from Discord"}`))
				return
			}

			var apiToken string
			if !exists {
				// Create user
				apiToken = utils.RandString(128)
				_, err = state.Pool.Exec(
					state.Context,
					"INSERT INTO users (user_id, api_token, username, staff, developer, certified) VALUES ($1, $2, $3, false, false, false)",
					user.ID,
					apiToken,
					user.Username,
				)

				if err != nil {
					state.Logger.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":true,"message":"Failed to create user"}`))
					return
				}
			} else {
				// Update user
				_, err = state.Pool.Exec(
					state.Context,
					"UPDATE users SET username = $1 WHERE user_id = $2",
					user.Username,
					user.ID,
				)

				if err != nil {
					state.Logger.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":true,"message":"Failed to update user"}`))
					return
				}

				// Get API token
				var tokenStr pgtype.Text

				err = state.Pool.QueryRow(state.Context, "SELECT api_token FROM users WHERE user_id = $1", user.ID).Scan(&tokenStr)

				if err != nil {
					state.Logger.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":true,"message":"Failed to get API token"}`))
					return
				}

				apiToken = tokenStr.String
			}

			// Check if user is banned from main server (TODO, not yet implemented)

			// Create authUser and send
			var authUser types.AuthUser = types.AuthUser{
				User:        discordUser,
				AccessToken: token.AccessToken,
				Token:       apiToken,
			}

			bytes, err := json.Marshal(authUser)

			if err != nil {
				state.Logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":true,"message":"Failed to marshal auth info"}`))
				return
			}

			w.Write(bytes)
		})

	})
}
