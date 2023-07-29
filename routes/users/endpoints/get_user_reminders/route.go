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
	silverpeltColsArr = utils.GetCols(types.Reminder{})
	silverpeltCols    = strings.Join(silverpeltColsArr, ",")
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
	rows, err := state.Pool.Query(d.Context, "SELECT "+silverpeltCols+" FROM silverpelt WHERE user_id = $1", id)

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
		// Try resolving the bot from discord API
		var resolvedBot types.ResolvedReminderBot
		bot, err := dovewing.GetUser(d.Context, reminder.BotID, state.DovewingPlatformDiscord)

		if err != nil {
			resolvedBot = types.ResolvedReminderBot{
				Name:   "Unknown",
				Avatar: "https://cdn.discordapp.com/embed/avatars/0.png",
			}
		} else {
			resolvedBot = types.ResolvedReminderBot{
				Name:   bot.Username,
				Avatar: bot.Avatar,
			}
		}

		reminders[i].ResolvedBot = resolvedBot
	}

	reminderList := types.ReminderList{
		Reminders: reminders,
	}

	return uapi.HttpResponse{
		Json: reminderList,
	}
}
