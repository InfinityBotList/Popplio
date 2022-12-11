package get_special_login_resp

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"popplio/state"
	"popplio/utils"
	"strings"
	"time"

	"github.com/infinitybotlist/eureka/crypto"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/cosmog",
		OpId:        "get_special_login_resp",
		Summary:     "Special Login Handler",
		Description: "This endpoint is used to respond to a special login. It then spawns the task such as data requests etc.",
		Tags:        []string{api.CurrentTag},
		Resp:        "[Redirect+Task Creation]",
	})
}

func Route(d api.RouteData, r *http.Request) {
	stateQuery := r.URL.Query().Get("state")

	// Get act from redis
	act, err := state.Redis.Get(d.Context, "spec:"+stateQuery).Result()

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
		return
	}

	// Decode act using json
	var action assets.Action

	err = json.Unmarshal([]byte(act), &action)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
		return
	}

	// Check time
	if time.Since(action.Time) > 3*time.Minute {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state (too old)",
		}
		return
	}

	// Check code with discords api
	data := url.Values{}

	data.Set("client_id", os.Getenv("KEY_ESCROW_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("KEY_ESCROW_CLIENT_SECRET"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("redirect_uri", os.Getenv("KEY_ESCROW_REDIRECT_URL"))

	response, err := http.PostForm("https://discord.com/api/oauth2/token", data)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	var token struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	err = json.Unmarshal(body, &token)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	state.Logger.Info(token)

	if !strings.Contains(token.Scope, "identify") {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid scope: scope contain identify, is currently " + token.Scope,
		}
		return
	}

	// Get user info
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: time.Second * 10}

	response, err = client.Do(req)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	var user assets.InternalOauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	if action.TID != "" {
		// Validate that they actually own this bot
		isOwner, err := utils.IsBotOwner(d.Context, user.ID, action.TID)

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		if !isOwner {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "You do not own the bot you are trying to manage",
			}
			return
		}
	}

	switch action.Action {
	// Data request
	case "dr":
		taskId := crypto.RandString(196)

		err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go assets.DataTask(taskId, user.ID, remoteIp[0], false)

		d.Resp <- api.HttpResponse{
			Redirect: os.Getenv("BOTLIST_APP") + "/data/confirm?tid=" + taskId + "&user=" + base64.URLEncoding.EncodeToString(body) + "&act=" + action.Action,
		}
		return
	// Data deletion request
	case "ddr":
		taskId := crypto.RandString(196)

		err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go assets.DataTask(taskId, user.ID, remoteIp[0], true)
		d.Resp <- api.HttpResponse{
			Redirect: os.Getenv("BOTLIST_APP") + "/data/confirm?tid=" + taskId + "&user=" + base64.URLEncoding.EncodeToString(body) + "&act=" + action.Action,
		}
		return
	// Reset token for users
	case "rtu":
		var token string
		token = crypto.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		d.Resp <- api.HttpResponse{
			Data: "Your new API token is: " + token + "\n\nThank you and have a nice day ;)",
		}
		return

	case "rtb":
		if action.TID == "" {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
			return
		}

		token := crypto.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET api_token = $1 WHERE bot_id = $2", token, action.TID)

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		d.Resp <- api.HttpResponse{
			Data: "Your new API token is: " + token + "\n\nThank you and have a nice day ;)",
		}
		return
	// Bot webhook secret
	case "bwebsec":
		if action.TID == "" {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
			return
		}

		if action.Ctx == "" {
			// We want to unset webhook secret
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET webhook_secret = NULL WHERE bot_id = $1", action.TID)

			if err != nil {
				d.Resp <- api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			d.Resp <- api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully unset webhook secret",
			}
			return
		} else {
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET webhook_secret = $1 WHERE bot_id = $2", action.Ctx, action.TID)

			if err != nil {
				d.Resp <- api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			d.Resp <- api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully set webhook secret",
			}
		}
	default:
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}
}
