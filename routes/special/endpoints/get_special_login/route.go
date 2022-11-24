package get_special_login

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/login/{act}",
		OpId:        "get_special_login",
		Summary:     "Special Login",
		Description: "This endpoint is used for special login actions. For example, data requests/deletions and regenerating tokens",
		Tags:        []string{api.CurrentTag},
		Resp:        "[Redirect]",
	})
}

func Route(d api.RouteData, r *http.Request) {
	cliId := os.Getenv("CLIENT_ID")
	redirectUrl := os.Getenv("REDIRECT_URL")

	tid := r.URL.Query().Get("tid")
	var tidInt int64
	var err error
	if tid != "" {
		tidInt, err = strconv.ParseInt(tid, 10, 64)

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "Invalid tid",
			}
			return
		}
	}

	var act = assets.Action{
		Action: chi.URLParam(r, "act"),
		Ctx:    r.URL.Query().Get("ctx"),
		Time:   time.Now(),
		TID:    tidInt,
	}

	// Encode act using gob
	var b bytes.Buffer
	e := gob.NewEncoder(&b)

	err = e.Encode(act)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   "Internal Server Error",
		}
		return
	}

	encPayload := base64.URLEncoding.EncodeToString(b.Bytes())

	d.Resp <- api.HttpResponse{
		Redirect: "https://discord.com/api/oauth2/authorize?client_id=" + cliId + "&scope=identify&response_type=code&redirect_uri=" + redirectUrl + "&state=" + encPayload,
	}
}
