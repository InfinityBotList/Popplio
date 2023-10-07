package get_servers_index

import (
	"context"
	"net/http"
	"strings"
	"time"

	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
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
	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "indexcache:servers").Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	listIndex := types.ListIndexServer{}

	// Certified Bots
	certRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Certified, err = processRow(d.Context, certRows)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Premium Bots
	premRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE premium = true ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Premium, err = processRow(d.Context, premRows)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Most Viewed Bots
	mostViewedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.MostViewed, err = processRow(d.Context, mostViewedRows)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Recently Added Bots
	recentlyAddedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.RecentlyAdded, err = processRow(d.Context, recentlyAddedRows)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Top Voted Bots
	topVotedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE type = 'approved' OR type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.TopVoted, err = processRow(d.Context, topVotedRows)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json:      listIndex,
		CacheKey:  "indexcache:servers",
		CacheTime: 3 * time.Minute,
	}
}

func processRow(ctx context.Context, rows pgx.Rows) ([]types.IndexServer, error) {
	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		return nil, err
	}

	for i := range servers {
		var code string

		err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", servers[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return nil, err
		}

		servers[i].Vanity = code
		servers[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeServers, servers[i].ServerID)
	}

	return servers, nil
}
