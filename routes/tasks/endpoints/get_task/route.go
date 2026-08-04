// Package get_task implements GET /users/{id}/tasks/{tid} — "Get Task".
//
// Gets a task. Returns the task data if this is successful
package get_task

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
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
			{
				Name:        "task_key",
				Description: "The task key if required. This is used to authenticate the request.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Task{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Check that the user owns the task
	taskId := chi.URLParam(r, "tid")
	userId := chi.URLParam(r, "id")

	if taskId == "" {
		return resp.BadRequest("task id is required")
	}

	// Delete expired tasks first
	_, err := state.Pool.Exec(d.Context, "DELETE FROM tasks WHERE created_at + expiry < NOW()")

	if err != nil {
		return resp.Err("Failed to delete expired tasks [db delete]", err)
	}

	row, err := state.Pool.Query(d.Context, "SELECT "+taskColsStr+" FROM tasks WHERE task_id = $1", taskId)

	if err != nil {
		return resp.Err("Failed to fetch task [db fetch]", err)
	}

	task, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Task])

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.NotFound("Task not found")
	}

	if err != nil {
		return resp.Err("Failed to fetch task [db fetch]", err)
	}

	if task.TaskKey.Valid {
		if task.TaskKey.String != r.URL.Query().Get("task_key") {
			return resp.Unauthorized("Invalid task key")
		}
	}

	if task.AllowUnauthenticated {
		d.Auth.ID = userId
	} else if d.Auth.ID == "" {
		return resp.Unauthorized("You must be authenticated to access this task")
	}

	if task.ForUser.Valid {
		if task.ForUser.String != d.Auth.ID {
			return resp.Forbidden("This task is not owned by your user account!")
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   task,
	}
}
