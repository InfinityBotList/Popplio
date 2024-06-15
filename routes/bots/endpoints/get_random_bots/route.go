package get_random_bots

import (
	"net/http"
	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Random Bots",
		Description: "Returns a list of bots from the database in random order",
		Resp: types.RandomBots{
			Bots: []types.IndexBot{},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY RANDOM() LIMIT 6")

	if err != nil {
		state.Logger.Error("Error while getting random bots [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error("Error while getting random bots [collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Set the user for each bot
	for i := range bots {
		err := assets.ResolveIndexBot(d.Context, &bots[i])

		if err != nil {
			state.Logger.Error("Error resolving indexbot", zap.Error(err), zap.String("botID", bots[i].BotID))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "An error occurred while resolving index bot: " + err.Error() + " botID: " + bots[i].BotID},
			}
		}
	}

	return uapi.HttpResponse{
		Json: types.RandomBots{
			Bots: bots,
		},
	}
}
