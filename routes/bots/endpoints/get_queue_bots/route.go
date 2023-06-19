package get_queue_bots

import (
	"net/http"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	queueBotColsArr = utils.GetCols(types.QueueBot{})
	queueBotCols    = strings.Join(queueBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Queue Bots",
		Description: "Gets queued bots on the list. Returns a set of ``QueueBot`` objects",
		Resp:        types.QueueBots{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+queueBotCols+" FROM bots WHERE type = 'pending' ORDER BY created_at")

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var bots []types.QueueBot

	err = pgxscan.ScanAll(&bots, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Set the user for each bot
	for i, bot := range bots {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.DovewingPlatformDiscord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].User = botUser

		if bots[i].ClaimedByID.Valid {
			claimedByUser, err := dovewing.GetUser(d.Context, bots[i].ClaimedByID.String, state.DovewingPlatformDiscord)

			if err != nil {
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			bots[i].ClaimedBy = claimedByUser
		}
	}

	return uapi.HttpResponse{
		Json: types.QueueBots{
			Bots: bots,
		},
	}
}
