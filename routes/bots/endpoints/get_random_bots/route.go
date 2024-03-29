package get_random_bots

import (
	"net/http"
	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
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
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY RANDOM() LIMIT 3")

	if err != nil {
		state.Logger.Error("Error while getting random bots [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error("Error while getting random bots [collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range bots {
		botUser, err := dovewing.GetUser(d.Context, bots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Error while getting random bots [dovewing]", zap.Error(err), zap.String("bot_id", bots[i].BotID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].User = botUser

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", bots[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error("Error while getting random bots [vanity]", zap.Error(err), zap.String("bot_id", bots[i].BotID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].Vanity = code
		bots[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeBots, bots[i].BotID)
	}

	return uapi.HttpResponse{
		Json: types.RandomBots{
			Bots: bots,
		},
	}
}
