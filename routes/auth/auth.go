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
	"github.com/jackc/pgx/v5/pgtype"
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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				resp <- types.HttpResponse{
					Data: clientInfo,
				}
			}()

			utils.Respond(ctx, w, resp)
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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				redirectUri := r.URL.Query().Get("redirect_uri")
				if !slices.Contains(allowedRedirectURLs, redirectUri) {
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Malformed redirect_uri"}`,
						Status: http.StatusBadRequest,
					}
					return
				}

				if redirectUri == allowedRedirectURLs[0] {
					if r.Header.Get("Wistala-Server") != os.Getenv("DEV_WISTALA_SERVER_SECRET") {
						resp <- types.HttpResponse{
							Data:   `{"error":true,"message":"This endpoint is not meant to be used by you"}`,
							Status: http.StatusForbidden,
						}
						return
					}
				} else {
					if r.Header.Get("Wistala-Server") != os.Getenv("WISTALA_SERVER_SECRET") {
						resp <- types.HttpResponse{
							Data:   `{"error":true,"message":"This endpoint is not meant to be used by you"}`,
							Status: http.StatusForbidden,
						}
						return
					}
				}

				code := r.URL.Query().Get("code")

				if code == "" {
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"No code provided"}`,
						Status: http.StatusBadRequest,
					}
					return
				}

				if state.Redis.Exists(ctx, "codecache:"+code).Val() == 1 {
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Code has been used before and is as such invalid"}`,
						Status: http.StatusBadRequest,
					}
					return
				}

				state.Redis.Set(ctx, "codecache:"+code, "0", 5*time.Minute)

				httpResp, err := http.PostForm("https://discord.com/api/v10/oauth2/token", url.Values{
					"client_id":     {os.Getenv("CLIENT_ID")},
					"client_secret": {os.Getenv("CLIENT_SECRET")},
					"grant_type":    {"authorization_code"},
					"code":          {code},
					"redirect_uri":  {redirectUri},
					"scope":         {"identify"},
				})

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to get token from Discord"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				defer httpResp.Body.Close()

				body, err := io.ReadAll(httpResp.Body)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to read token response from Discord"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				var token struct {
					AccessToken string `json:"access_token"`
				}

				err = json.Unmarshal(body, &token)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to parse token response from Discord"}`,
						Status: http.StatusBadRequest,
					}
					return
				}

				if token.AccessToken == "" {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"No access token provided by Discord"}`,
						Status: http.StatusBadRequest,
					}
					return
				}

				cli := &http.Client{}

				req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to create request to Discord to fetch user info"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				req.Header.Set("Authorization", "Bearer "+token.AccessToken)

				httpResp, err = cli.Do(req)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to get user info from Discord"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				defer httpResp.Body.Close()

				body, err = io.ReadAll(httpResp.Body)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to read user info response from Discord"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				var user OauthUser

				err = json.Unmarshal(body, &user)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to parse user info response from Discord"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				if user.ID == "" {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"No user ID provided by Discord. Invalid token?"}`,
						Status: http.StatusBadRequest,
					}
					return
				}

				// Check if user exists on database
				var exists bool

				err = state.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", user.ID).Scan(&exists)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to check if user exists on database"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				discordUser, err := utils.GetDiscordUser(user.ID)

				if err != nil {
					state.Logger.Error(err)
					resp <- types.HttpResponse{
						Data:   `{"error":true,"message":"Failed to get user info from Discord"}`,
						Status: http.StatusInternalServerError,
					}
					return
				}

				var apiToken string
				if !exists {
					// Create user
					apiToken = utils.RandString(128)
					_, err = state.Pool.Exec(
						ctx,
						"INSERT INTO users (user_id, api_token, username, staff, developer, certified) VALUES ($1, $2, $3, false, false, false)",
						user.ID,
						apiToken,
						user.Username,
					)

					if err != nil {
						state.Logger.Error(err)
						resp <- types.HttpResponse{
							Data:   `{"error":true,"message":"Failed to create user on database"}`,
							Status: http.StatusInternalServerError,
						}
						return
					}
				} else {
					// Update user
					_, err = state.Pool.Exec(
						ctx,
						"UPDATE users SET username = $1 WHERE user_id = $2",
						user.Username,
						user.ID,
					)

					if err != nil {
						state.Logger.Error(err)
						resp <- types.HttpResponse{
							Data:   `{"error":true,"message":"Failed to update user on database"}`,
							Status: http.StatusInternalServerError,
						}
						return
					}

					// Get API token
					var tokenStr pgtype.Text

					err = state.Pool.QueryRow(ctx, "SELECT api_token FROM users WHERE user_id = $1", user.ID).Scan(&tokenStr)

					if err != nil {
						state.Logger.Error(err)
						resp <- types.HttpResponse{
							Data:   `{"error":true,"message":"Failed to get API token from database"}`,
							Status: http.StatusInternalServerError,
						}
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

				resp <- types.HttpResponse{
					Json: authUser,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

	})
}
