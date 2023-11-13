package get_task

import (
	"errors"
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	taskColsArr = db.GetCols(types.Task{})
	taskColsStr = strings.Join(taskColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Task",
		Description: "Gets a task. Returns the task data if this is successful",
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
		Resp: types.Task{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Check that the user owns the task
	taskId := chi.URLParam(r, "tid")

	if taskId == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "task id is required"},
		}
	}

	// Delete expired tasks first
	_, err := state.Pool.Exec(d.Context, "DELETE FROM tasks WHERE created_at + expiry < NOW()")

	if err != nil {
		state.Logger.Error("Failed to delete expired tasks [db delete]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	row, err := state.Pool.Query(d.Context, "SELECT "+taskColsStr+" FROM tasks WHERE task_id = $1", taskId)

	if err != nil {
		state.Logger.Error("Failed to fetch task [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	task, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Task])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Task not found"},
		}
	}

	if err != nil {
		state.Logger.Error("Failed to fetch task [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if task.ForUser.Valid {
		if task.ForUser.String != d.Auth.ID {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "This task is not owned by your user account!"},
			}
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   task,
	}
}
