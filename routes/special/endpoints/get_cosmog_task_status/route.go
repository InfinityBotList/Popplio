package get_cosmog_task_status

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

func Docs() {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/cosmog/tasks/{tid}",
		OpId:        "get_cosmog_task_status",
		Summary:     "Special Login Task View",
		Description: "Shows the status of a task that has been started by a special login.",
		Tags:        []string{api.CurrentTag},
		Resp:        "[HTML]",
	})
}

func Route(d api.RouteData, r *http.Request) {
	var user assets.InternalOauthUser

	tid := chi.URLParam(r, "tid")

	if tid == "" {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid task id",
		}
		return
	}

	userStr := r.URL.Query().Get("n")

	if userStr == "" {
		user = assets.InternalOauthUser{
			ID:       "Unknown",
			Username: "Unknown",
			Disc:     "0000",
		}
	} else {
		body, err := base64.URLEncoding.DecodeString(userStr)

		if err != nil {
			d.Resp <- types.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}

		err = json.Unmarshal(body, &user)

		if err != nil {
			d.Resp <- types.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
			return
		}
	}

	user.TID = tid

	t, err := template.ParseFiles("html/taskpage.html")

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	templWriter := bytes.NewBuffer([]byte{})

	t.Execute(templWriter, user)

	d.Resp <- types.HttpResponse{
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Bytes: templWriter.Bytes(),
	}
}
