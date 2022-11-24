package get_special_login_resp

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
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

	// Hex decode act
	hexedAct, err := base64.URLEncoding.DecodeString(stateQuery)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
		return
	}

	// Decode act using gob
	var b bytes.Buffer
	b.Write(hexedAct)

	action := assets.Action{}

	dg := gob.NewDecoder(&b)

	err = dg.Decode(&action)

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

	data.Set("client_id", os.Getenv("CLIENT_ID"))
	data.Set("client_secret", os.Getenv("CLIENT_SECRET"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("redirect_uri", os.Getenv("REDIRECT_URL"))

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

	taskId := utils.RandString(196)

	err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

	if action.Action == "dr" {
		go assets.DataTask(taskId, user.ID, remoteIp[0], false)
	} else if action.Action == "ddr" {
		go assets.DataTask(taskId, user.ID, remoteIp[0], true)
	} else if action.Action == "rt" {
		if action.Ctx == "" {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No action context found",
			}
			return
		}

		if action.Ctx == "@me" {
			token := utils.RandString(128)

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
		} else {
			if action.TID == 0 {
				d.Resp <- api.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "No target id set",
				}
				return
			}

			token := utils.RandString(128)

			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET api_token = $1 WHERE bot_id = $2", token, action.TID)

			if err != nil {
				d.Resp <- api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}
		}
	} else {
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	d.Resp <- api.HttpResponse{
		Redirect: os.Getenv("BOTLIST_APP") + "/data/confirm?tid=" + taskId + "&user=" + base64.URLEncoding.EncodeToString(body) + "&act=" + action.Action,
	}

}
