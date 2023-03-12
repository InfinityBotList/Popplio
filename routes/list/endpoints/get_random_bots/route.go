package get_random_bots

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/doclib"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

type RandomBotResponse struct {
	Bots  []types.IndexBot `json:"bots"`
	Count int              `json:"count"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Random Bots",
		Description: "Returns a list of bots from the database in random order",
		Resp: RandomBotResponse{
			Bots: []types.IndexBot{},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY RANDOM() LIMIT 3")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var indexBots = []types.IndexBot{}

	err = pgxscan.ScanAll(&indexBots, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i, bot := range indexBots {
		botUser, err := utils.GetDiscordUser(d.Context, bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		indexBots[i].User = botUser
	}

	return api.HttpResponse{
		Json: RandomBotResponse{
			Bots:  indexBots,
			Count: len(indexBots),
		},
	}

}
