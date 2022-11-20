package get_cosmog_task_tid

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/cosmog/tasks/{tid}.arceus",
		OpId:        "get_cosmog_task_tid",
		Summary:     "Special Login Task View JSON",
		Description: "Returns the status of a task as a arbitary json.",
		Tags:        []string{api.CurrentTag},
		Resp:        "[JSON]",
	})
}

func Route(d api.RouteData, r *http.Request) {
	tid := chi.URLParam(r, "tid")

	if tid == "" {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid task id",
		}
		return
	}

	task, err := state.Redis.Get(d.Context, tid).Result()

	if err == redis.Nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusNotFound,
			Data:   "Task not found",
		}
		return
	}

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
		return
	}

	d.Resp <- types.HttpResponse{
		Data: task,
	}

}
