package get_bots_index

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	indexPackColsArr = db.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bots Index",
		Description: "Gets the index of the bot-side of the list. Returns a ``ListIndexBot`` object",
		Resp:        types.ListIndexBot{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listIndex := types.ListIndexBot{}

	// Certified Bots
	certRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting certified bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Certified, err = processRow(d.Context, certRows)
	if err != nil {
		state.Logger.Error("Error while processing certified bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Premium Bots
	premRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE premium = true ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting premium bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Premium, err = processRow(d.Context, premRows)
	if err != nil {
		state.Logger.Error("Error while processing premium bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Most Viewed Bots
	mostViewedRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting most viewed bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.MostViewed, err = processRow(d.Context, mostViewedRows)
	if err != nil {
		state.Logger.Error("Error while processing most viewed bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Recently Added Bots
	recentlyAddedRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting recently added bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.RecentlyAdded, err = processRow(d.Context, recentlyAddedRows)
	if err != nil {
		state.Logger.Error("Error while processing recently added bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Top Voted Bots
	topVotedRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting top voted bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.TopVoted, err = processRow(d.Context, topVotedRows)
	if err != nil {
		state.Logger.Error("Error while processing top voted bots", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Packs
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error("Error while getting packs [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	listIndex.Packs, err = pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBotPack])

	if err != nil {
		state.Logger.Error("Error while getting packs [collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: listIndex,
	}
}

func processRow(ctx context.Context, rows pgx.Rows) ([]types.IndexBot, error) {
	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		return nil, err
	}

	for i := range bots {
		botUser, err := dovewing.GetUser(ctx, bots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, err
		}

		bots[i].User = botUser

		var code string

		err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", bots[i].VanityRef).Scan(&code)

		if err != nil {
			return nil, fmt.Errorf("error while getting vanity: %w", err)
		}

		bots[i].Vanity = code
		bots[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeBots, bots[i].BotID)
	}

	return bots, nil
}
