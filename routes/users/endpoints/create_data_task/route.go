package create_data_task

import (
	"net/http"
	"popplio/api"
	"popplio/routes/users/endpoints/create_data_task/assets"
	"popplio/state"
	"popplio/types"
	"strings"
	"time"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/eureka/crypto"
)

type DataTaskResponse struct {
	TaskID string `json:"task_id"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Data Task",
		Description: "Creates a data task for a user (delete or request). Returns the task id if this is successful.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "delete",
				Description: "Whether we should do a Data Deletion or a Data Request",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: DataTaskResponse{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	reqType := r.URL.Query().Get("delete")

	if reqType != "true" && reqType != "false" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "delete must be ether 'true' or 'false'", Error: true},
		}
	}

	taskId := crypto.RandString(196)

	err := state.Redis.Set(d.Context, "data:"+taskId+"_status", "[]", time.Hour*4).Err()

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	err = state.Redis.Set(d.Context, "data:"+taskId+"_owner", d.Auth.ID, time.Hour*4).Err()

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

	go assets.DataTask("data:"+taskId, d.Auth.ID, remoteIp[0], reqType == "true")

	return api.HttpResponse{
		Json: DataTaskResponse{TaskID: taskId},
	}
}
