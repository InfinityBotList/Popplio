// Package delete_all_user_alerts implements DELETE /users/{id}/alerts —
// "Delete All User Alerts".
//
// Deletes all user alerts. Returns 204 on success
package delete_all_user_alerts

import (
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete All User Alerts",
		Description: "Deletes all user alerts. Returns 204 on success",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	_, err := state.Pool.Exec(d.Context, "DELETE FROM alerts WHERE user_id = $1", d.Auth.ID)

	if err != nil {
		return resp.Err("Failed to delete alerts", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
