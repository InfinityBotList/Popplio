package put_user

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/ratelimit"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/exp/slices"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var allowedRedirectURLs = []string{
	"http://localhost:3000/sauron",               // DEV
	"https://reedwhisker.infinitybots.gg/sauron", // PROD
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PUT",
		Path:        "/users",
		OpId:        "put_user",
		Summary:     "Login User",
		Description: "Takes in a ``code`` query parameter and returns a user ``token``. **Cannot be used outside of the site for security reasons but documented in case we wish to allow its use in the future.**",
		Tags:        []string{api.CurrentTag},
		Req:         AuthorizeRequest{},
		Resp:        types.AuthUser{},
	})
}

type AuthorizeRequest struct {
	Code        string `json:"code" validate:"required,min=5"`
	RedirectURI string `json:"redirect_uri" validate:"required"`
	Nonce       string `json:"nonce" validate:"required"` // Just to identify and block older clients from vulns
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "login",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var req AuthorizeRequest

	hresp, ok := api.MarshalReqWithHeaders(r, &req, limit.Headers())

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(req)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(api.BlankMap, errors)
	}

	if !slices.Contains(allowedRedirectURLs, req.RedirectURI) {
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Malformed redirect_uri"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if req.Nonce != "sauron" {
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Your client is outdated and is not supported. Please update your client."}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if req.Code == "" {
		return api.HttpResponse{
			Data:    `{"error":true,"message":"No code provided"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if len(req.Code) < 5 {
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Code too short. Retry login?"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if state.Redis.Exists(d.Context, "codecache:"+req.Code).Val() == 1 {
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Code has been used before and is as such invalid"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	state.Redis.Set(d.Context, "codecache:"+req.Code, "0", 5*time.Minute)

	httpResp, err := http.PostForm("https://discord.com/api/v10/oauth2/token", url.Values{
		"client_id":     {os.Getenv("CLIENT_ID")},
		"client_secret": {os.Getenv("CLIENT_SECRET")},
		"grant_type":    {"authorization_code"},
		"code":          {req.Code},
		"redirect_uri":  {req.RedirectURI},
		"scope":         {"identify"},
	})

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to get token from Discord"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to read token response from Discord"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}

	err = json.Unmarshal(body, &token)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to parse token response from Discord"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if token.AccessToken == "" {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"No access token provided by Discord"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	cli := &http.Client{}

	var httpReq *http.Request
	httpReq, err = http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to create request to Discord to fetch user info"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	httpResp, err = cli.Do(httpReq)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to get user info from Discord"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	defer httpResp.Body.Close()

	body, err = io.ReadAll(httpResp.Body)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to read user info response from Discord"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	var user types.OauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to parse user info response from Discord"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	if user.ID == "" {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"No user ID provided by Discord. Invalid token?"}`,
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	// Check if user exists on database
	var exists bool

	err = state.Pool.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", user.ID).Scan(&exists)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to check if user exists on database"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	discordUser, err := utils.GetDiscordUser(user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Data:    `{"error":true,"message":"Failed to get user info from Discord"}`,
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	var apiToken string
	if !exists {
		// Create user
		apiToken = crypto.RandString(128)
		_, err = state.Pool.Exec(
			d.Context,
			"INSERT INTO users (user_id, api_token, username, staff, developer, certified) VALUES ($1, $2, $3, false, false, false)",
			user.ID,
			apiToken,
			user.Username,
		)

		if err != nil {
			state.Logger.Error(err)
			return api.HttpResponse{
				Data:    `{"error":true,"message":"Failed to create user on database"}`,
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
			}
		}
	} else {
		// Update user
		_, err = state.Pool.Exec(
			d.Context,
			"UPDATE users SET username = $1 WHERE user_id = $2",
			user.Username,
			user.ID,
		)

		if err != nil {
			state.Logger.Error(err)
			return api.HttpResponse{
				Data:    `{"error":true,"message":"Failed to update user on database"}`,
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
			}
		}

		// Get API token
		var tokenStr pgtype.Text

		err = state.Pool.QueryRow(d.Context, "SELECT api_token FROM users WHERE user_id = $1", user.ID).Scan(&tokenStr)

		if err != nil {
			state.Logger.Error(err)
			return api.HttpResponse{
				Data:    `{"error":true,"message":"Failed to get API token from database"}`,
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
			}
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

	return api.HttpResponse{
		Json:    authUser,
		Headers: limit.Headers(),
	}
}