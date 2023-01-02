package get_cosmog_task_tid

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Path:        "/cosmog/tasks/{tid}.arceus",
		Summary:     "Special Login Task View JSON",
		Description: "Returns the status of a task as a arbitary json.",
		Resp:        "[JSON]",
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	tid := chi.URLParam(r, "tid")

	if tid == "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid task id",
		}
	}

	task, err := state.Redis.Get(d.Context, tid).Result()

	if err == redis.Nil {
		return api.HttpResponse{
			Status: http.StatusNotFound,
			Data:   "Task not found",
		}
	}

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	return api.HttpResponse{
		Data: task,
	}
}
