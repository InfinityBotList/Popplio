package get_special_login_resp

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"
	"time"
)

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
	act := r.URL.Query().Get("state")

	// Split act and hmac
	actSplit := strings.Split(act, ".")

	if len(actSplit) != 3 {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
		return
	}

	// Check hmac
	h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

	h.Write([]byte(actSplit[0] + "@" + actSplit[2]))

	hmacData := hex.EncodeToString(h.Sum(nil))

	if hmacData != actSplit[1] {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
		return
	}

	// Check time
	ctime, err := strconv.ParseInt(actSplit[0], 10, 64)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
		return
	}

	if time.Now().Unix()-ctime > 300 {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state, HMAC is too old",
		}
		return
	}

	// Remove out the actual action
	act = actSplit[2]

	// Check code with discords api
	data := url.Values{}

	data.Set("client_id", os.Getenv("CLIENT_ID"))
	data.Set("client_secret", os.Getenv("CLIENT_SECRET"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("redirect_uri", os.Getenv("REDIRECT_URL"))

	response, err := http.PostForm("https://discord.com/api/oauth2/token", data)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		d.Resp <- types.HttpResponse{
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
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	state.Logger.Info(token)

	if !strings.Contains(token.Scope, "identify") {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid scope: scope contain identify, is currently " + token.Scope,
		}
		return
	}

	// Get user info
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: time.Second * 10}

	response, err = client.Do(req)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	var user assets.InternalOauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	taskId := utils.RandString(196)

	err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

	if act == "dr" {
		go assets.DataTask(taskId, user.ID, remoteIp[0], false)
	} else if act == "ddr" {
		go assets.DataTask(taskId, user.ID, remoteIp[0], true)
	} else if act == "gettoken" {
		token := utils.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

		if err != nil {
			d.Resp <- types.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		d.Resp <- types.HttpResponse{
			Data: token,
		}
		return

	} else {
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	d.Resp <- types.HttpResponse{
		Redirect: os.Getenv("BOTLIST_APP") + "/data/confirm?tid=" + taskId + "&user=" + base64.URLEncoding.EncodeToString(body) + "&act=" + act,
	}

}
