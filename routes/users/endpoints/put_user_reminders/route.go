package put_user_reminders

import (
	"net/http"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Reminder",
		Description: "Creates a new user reminders of a bot deleting existing ones for the bot",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "Bot ID to add a reminder of",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReminderList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Delete old
	state.Pool.Exec(d.Context, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", d.Auth.ID, id)

	// Add new
	_, err = state.Pool.Exec(d.Context, "INSERT INTO silverpelt (user_id, bot_id) VALUES ($1, $2)", d.Auth.ID, id)

	if err != nil {
		state.Logger.Error("Error adding reminder: ", err)
		return api.DefaultResponse(http.StatusBadRequest)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
