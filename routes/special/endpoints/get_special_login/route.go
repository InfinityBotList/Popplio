package get_special_login

import (
	"bytes"
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"popplio/state"
	"popplio/utils"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/login-cosmog",
		OpId:        "get_special_login",
		Summary:     "Special Login",
		Description: "This endpoint is used for special login actions. For example, data requests/deletions and regenerating tokens",
		Tags:        []string{api.CurrentTag},
		Resp:        assets.Redirect{},
	})
}

func Route(d api.RouteData, r *http.Request) {
	// Read assets.Action to get the action
	var action assets.Action

	err := json.NewDecoder(r.Body).Decode(&action)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid request body",
		}
		return
	}

	if action.Action == "" {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid action",
		}
		return
	}

	action.Time = time.Now()

	cliId := os.Getenv("KEY_ESCROW_CLIENT_ID")
	redirectUrl := os.Getenv("KEY_ESCROW_REDIRECT_URL")

	tid := r.URL.Query().Get("tid")
	if action.TID != "" {
		_, err := strconv.ParseInt(tid, 10, 64)

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "Invalid tid",
			}
			return
		}
	}

	// Encode act using gob
	var b bytes.Buffer
	e := json.NewEncoder(&b)

	err = e.Encode(action)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   "Internal Server Error",
		}
		return
	}

	// Store in redis
	stateTok := utils.RandString(64)
	err = state.Redis.Set(d.Context, "spec:"+stateTok, b.Bytes(), 5*time.Minute).Err()

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   "Internal Server Error",
		}
		return
	}

	d.Resp <- api.HttpResponse{
		Json: assets.Redirect{
			Redirect: "https://discord.com/api/oauth2/authorize?client_id=" + cliId + "&scope=identify&response_type=code&redirect_uri=" + redirectUrl + "&state=" + stateTok,
		},
	}
}
