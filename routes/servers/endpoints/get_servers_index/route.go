package get_servers_index

import (
	"context"
	"net/http"
	"strings"

	"popplio/db"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexServersColsArr = db.GetCols(types.IndexServer{})
	indexServersCols    = strings.Join(indexServersColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Servers Index",
		Description: "Gets the index of the server-side of the list. Returns a ``ListIndexServer`` object",
		Resp:        types.ListIndexServer{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listIndex := types.ListIndexServer{}

	// Certified Servers
	certRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting certified servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Certified, err = processRow(d.Context, certRows)
	if err != nil {
		state.Logger.Error("Error while processing certified servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Premium Servers
	premRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE premium = true ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting premium servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Premium, err = processRow(d.Context, premRows)
	if err != nil {
		state.Logger.Error("Error while processing premium servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Most Viewed Servers
	mostViewedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting most viewed servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.MostViewed, err = processRow(d.Context, mostViewedRows)
	if err != nil {
		state.Logger.Error("Error while processing most viewed servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Recently Added Servers
	recentlyAddedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting recently added servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.RecentlyAdded, err = processRow(d.Context, recentlyAddedRows)
	if err != nil {
		state.Logger.Error("Error while processing recently added servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Top Voted Servers
	topVotedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'approved' OR type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error("Error while getting top voted servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.TopVoted, err = processRow(d.Context, topVotedRows)
	if err != nil {
		state.Logger.Error("Error while processing top voted servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: listIndex,
	}
}

func processRow(ctx context.Context, rows pgx.Rows) ([]types.IndexServer, error) {
	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		return nil, err
	}

	for i := range servers {
		err := assets.ResolveIndexServer(ctx, &servers[i])

		if err != nil {
			return nil, err
		}
	}

	return servers, nil
}
