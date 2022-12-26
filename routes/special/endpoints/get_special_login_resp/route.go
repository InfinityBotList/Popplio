package get_special_login_resp

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/infinitybotlist/eureka/crypto"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/exp/slices"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	stateQuery := r.URL.Query().Get("state")

	// Get act from redis
	act, err := state.Redis.Get(d.Context, "spec:"+stateQuery).Result()

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
	}

	state.Redis.Del(d.Context, "spec:"+stateQuery)

	// Decode act using json
	var action assets.Action

	err = json.Unmarshal([]byte(act), &action)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
	}

	// Check time
	if time.Since(action.Time) > 3*time.Minute {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state (too old)",
		}
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
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	var token struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	err = json.Unmarshal(body, &token)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	state.Logger.Info(token)

	if !strings.Contains(token.Scope, "identify") {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid scope: scope contain identify, is currently " + token.Scope,
		}
	}

	// Get user info
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: time.Second * 10}

	response, err = client.Do(req)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	var user types.OauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	if action.TID != "" {
		// Validate that they actually own this bot
		isOwner, err := utils.IsBotOwner(d.Context, user.ID, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		if !isOwner {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "You do not own the bot you are trying to manage",
			}
		}
	}

	switch action.Action {
	// Data request
	case "dr":
		taskId := crypto.RandString(196)

		err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go assets.DataTask(taskId, user.ID, remoteIp[0], false)

		return api.HttpResponse{
			Redirect: os.Getenv("BOTLIST_APP") + "/data/confirm?tid=" + taskId + "&user=" + base64.URLEncoding.EncodeToString(body) + "&act=" + action.Action,
		}
	// Data deletion request
	case "ddr":
		taskId := crypto.RandString(196)

		err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go assets.DataTask(taskId, user.ID, remoteIp[0], true)
		return api.HttpResponse{
			Redirect: os.Getenv("BOTLIST_APP") + "/data/confirm?tid=" + taskId + "&user=" + base64.URLEncoding.EncodeToString(body) + "&act=" + action.Action,
		}
	// Reset token for users
	case "rtu":
		var token string
		token = crypto.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Data: "Your new API token is: " + token + "\n\nThank you and have a nice day ;)",
		}
	// Reset token for bots
	case "rtb":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		token := crypto.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET api_token = $1 WHERE bot_id = $2", token, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Data: "Your new API token is: " + token + "\n\nThank you and have a nice day ;)",
		}
	// Bot webhook url update
	case "bweburl":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if action.Ctx == "" {
			// We want to unset webhook secret
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully unset webhook url",
			}
		} else {
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET webhook = $1 WHERE bot_id = $2", action.Ctx, action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully set webhook url",
			}
		}
	// Bot webhook secret update
	case "bwebsec":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if action.Ctx == "" {
			// We want to unset webhook secret
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET web_auth = NULL WHERE bot_id = $1", action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully unset webhook secret",
			}
		} else {
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET web_auth = $1 WHERE bot_id = $2", action.Ctx, action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully set webhook secret",
			}
		}
	// Delete the bot
	case "dbot":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		// Get main owner of bot
		var owner string

		err := state.Pool.QueryRow(d.Context, "SELECT owner FROM bots WHERE bot_id = $1", action.TID).Scan(&owner)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		// Check if user is main owner
		if owner != user.ID {
			return api.HttpResponse{
				Status: http.StatusForbidden,
				Data:   "You are not the main owner of this bot. Only main owners can delete bots",
			}
		}

		// Delete bot
		_, err = state.Pool.Exec(d.Context, "DELETE FROM bots WHERE bot_id = $1", action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Status: http.StatusOK,
			Data:   "Successfully deleted bot :)",
		}
	// Transfer bot ownership
	case "tbot":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		// Get main owner of bot
		var owner string

		err := state.Pool.QueryRow(d.Context, "SELECT owner FROM bots WHERE bot_id = $1", action.TID).Scan(&owner)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		// Check if user is main owner
		if owner != user.ID {
			return api.HttpResponse{
				Status: http.StatusForbidden,
				Data:   "You are not the main owner of this bot. Only main owners can transfer the ownership of bots",
			}
		}

		// Ensure new owner is currently an additional owner
		var additionalOwners []string

		err = state.Pool.QueryRow(d.Context, "SELECT additional_owners FROM bots WHERE bot_id = $1", action.TID).Scan(&additionalOwners)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		if !slices.Contains(additionalOwners, action.Ctx) {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "New owner is not currently an additional owner!",
			}
		}

		// Transfer ownership
		tr, err := state.Pool.Begin(d.Context)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		defer tr.Rollback(d.Context)

		_, err = tr.Exec(d.Context, "UPDATE bots SET owner = $1 WHERE bot_id = $2", action.Ctx, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		// Remove new owner from additional owners
		_, err = tr.Exec(d.Context, "UPDATE bots SET additional_owners = array_remove(additional_owners, $1) WHERE bot_id = $2", action.Ctx, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		// Add old owner to additional owners
		_, err = tr.Exec(d.Context, "UPDATE bots SET additional_owners = array_append(additional_owners, $1) WHERE bot_id = $2", owner, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		err = tr.Commit(d.Context)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Status: http.StatusOK,
			Data:   "Successfully transferred ownership of bot. The old owner (you!) is now an additional owner and the new owner is the main owner now.",
		}

	default:
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid action",
		}
	}
}
