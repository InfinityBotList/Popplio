package get_data_task

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type DataTask struct {
	Statuses []string `json:"statuses"`
	Output   string   `json:"output"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Data Task",
		Description: "Gets the data task. Returns the task data if this is successful",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tid",
				Description: "The task ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: DataTask{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Check that the user owns the task
	taskId := chi.URLParam(r, "tid")

	if taskId == "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "task id is required", Error: true},
		}
	}

	// Get the tasks status output
	statusesRaw := state.Redis.Get(d.Context, "data:"+taskId+"_status").Val()

	if statusesRaw == "" {
		return api.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Task has invalid status", Error: true},
		}
	}

	// Parse statuses
	var statuses []string

	err := json.Unmarshal([]byte(statusesRaw), &statuses)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	output := state.Redis.Get(d.Context, "data:"+taskId+"_out").Val()

	return api.HttpResponse{
		Status: http.StatusOK,
		Json: DataTask{
			Statuses: statuses,
			Output:   output,
		},
	}
}
