package get_user_reminders

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	silverpeltColsArr = utils.GetCols(types.Reminder{})
	silverpeltCols    = strings.Join(silverpeltColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}/reminders",
		OpId:        "get_user_reminders",
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
		Resp:     types.ReminderList{},
		Tags:     []string{api.CurrentTag},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var id = chi.URLParam(r, "id")

	// Fetch reminder from postgres
	rows, err := state.Pool.Query(d.Context, "SELECT "+silverpeltCols+" FROM silverpelt WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var reminders []types.Reminder

	pgxscan.ScanAll(&reminders, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if len(reminders) == 0 {
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	for i, reminder := range reminders {
		// Try resolving the bot from discord API
		var resolvedBot types.ResolvedReminderBot
		bot, err := utils.GetDiscordUser(reminder.BotID)

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

	d.Resp <- api.HttpResponse{
		Json: reminderList,
	}
}