package create_data_task

import (
	"net/http"
	"popplio/routes/users/endpoints/create_data_task/assets"
	"popplio/state"
	"popplio/types"
	"strings"
	"time"

	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const dataTaskExpiryTime = time.Hour * 1

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
		Resp: types.TaskCreateResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	reqType := r.URL.Query().Get("delete")

	if reqType != "true" && reqType != "false" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "delete must be ether 'true' or 'false'"},
		}
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Hour,
		MaxRequests: 50,
		Bucket:      "data_request",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error("Error while ratelimiting", zap.Error(err), zap.String("bucket", "data_request"))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	taskName := "data_request"

	if reqType == "true" {
		taskName = "data_delete"
	}

	remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

	taskKey := crypto.RandString(128)
	var taskId string

	allowUnauthenticated := (taskName == "data_delete") // Only data deletions need unauthenticated access to task data

	err = state.Pool.QueryRow(d.Context, "INSERT INTO tasks (task_name, task_key, for_user, expiry, output, allow_unauthenticated) VALUES ($1, $2, $3, $4, $5, $6) RETURNING task_id",
		taskName,
		taskKey,
		d.Auth.ID,
		dataTaskExpiryTime,
		map[string]any{
			"meta": map[string]any{
				"request_ip": remoteIp[0],
			},
		},
		allowUnauthenticated,
	).Scan(&taskId)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Error creating task:" + err.Error(),
			},
		}
	}

	go assets.DataTask(taskId, taskName, d.Auth.ID, remoteIp[0])

	return uapi.HttpResponse{
		Json: types.TaskCreateResponse{
			TaskID: taskId,
			TaskKey: pgtype.Text{
				Valid:  true,
				String: taskKey,
			},
			TaskName:             taskName,
			Expiry:               pgtype.Interval{Microseconds: int64(dataTaskExpiryTime / time.Microsecond)},
			AllowUnauthenticated: allowUnauthenticated,
		},
	}
}
