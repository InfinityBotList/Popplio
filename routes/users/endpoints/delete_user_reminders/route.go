package delete_user_reminders

import (
	"net/http"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return &docs.Doc{
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
		Resp: types.ReminderList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	var botId pgtype.Text

	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE "+constants.ResolveBotSQL, r.URL.Query().Get("bot_id")).Scan(&botId)

	if err != nil || !botId.Valid || botId.String == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Delete old
	state.Pool.Exec(d.Context, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", id, botId.String)

	return api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
