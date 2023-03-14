package get_user_reminders

import (
	"net/http"
	"strings"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"

	"github.com/georgysavva/scany/v2/pgxscan"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	// Fetch reminder from postgres
	rows, err := state.Pool.Query(d.Context, "SELECT "+silverpeltCols+" FROM silverpelt WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var reminders []types.Reminder

	pgxscan.ScanAll(&reminders, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if len(reminders) == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	for i, reminder := range reminders {
		// Try resolving the bot from discord API
		var resolvedBot types.ResolvedReminderBot
		bot, err := dovewing.GetDiscordUser(d.Context, reminder.BotID)

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

	return api.HttpResponse{
		Json: reminderList,
	}
}
