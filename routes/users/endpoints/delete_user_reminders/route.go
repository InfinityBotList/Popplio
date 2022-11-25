package delete_user_reminders

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "DELETE",
		Path:        "/users/{id}/reminder",
		OpId:        "delete_user_reminders",
		Summary:     "Delete User Reminders",
		Description: "Deletes a users reminders. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bot_id",
				Description: "Bot ID to delete a reminder of",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp:     types.ReminderList{},
		Tags:     []string{api.CurrentTag},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var id = chi.URLParam(r, "id")

	var botId pgtype.Text

	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", r.URL.Query().Get("bot_id")).Scan(&botId)

	if err != nil || !botId.Valid || botId.String == "" {
		state.Logger.Error("Error deleting reminder: ", err)
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	// Delete old
	state.Pool.Exec(d.Context, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", id, botId.String)

	d.Resp <- api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
