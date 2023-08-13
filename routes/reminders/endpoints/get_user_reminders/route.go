package get_user_reminders

import (
	"net/http"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
)

var (
	reminderColsArr = utils.GetCols(types.Reminder{})
	reminderCols    = strings.Join(reminderColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Reminders",
		Description: "Gets a users reminders",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReminderList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	// Fetch reminder from postgres
	rows, err := state.Pool.Query(d.Context, "SELECT "+reminderCols+" FROM user_reminders WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	reminders, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Reminder])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i, reminder := range reminders {
		// Try resolving the entity from discord API
		reminders[i].Resolved = &types.ResolvedReminder{
			Name:   "Unknown",
			Avatar: "https://cdn.discordapp.com/embed/avatars/0.png",
		}

		switch reminder.TargetType {
		case "bot":
			bot, err := dovewing.GetUser(d.Context, reminder.TargetID, state.DovewingPlatformDiscord)

			if err == nil {
				reminders[i].Resolved = &types.ResolvedReminder{
					Name:   bot.Username,
					Avatar: bot.Avatar,
				}
			}
		case "team":
			var name, avatar string

			err := state.Pool.QueryRow(d.Context, "SELECT name, avatar FROM teams WHERE id = $1", reminder.TargetID).Scan(&name, &avatar)

			if err == nil {
				reminders[i].Resolved = &types.ResolvedReminder{
					Name:   name,
					Avatar: avatar,
				}
			}
		}
	}

	reminderList := types.ReminderList{
		Reminders: reminders,
	}

	return uapi.HttpResponse{
		Json: reminderList,
	}
}
