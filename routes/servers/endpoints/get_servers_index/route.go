// Package get_servers_index implements GET /servers/@index — "Get Servers
// Index".
//
// Gets the index of the server-side of the list. Returns a `ListIndexServer`
// object
package get_servers_index

import (
	"context"
	"fmt"
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/routes/servers/assets"
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
	listIndex := types.ListIndexServer{}

	// Certified Servers
	certRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE state = 'public' AND type = 'certified' ORDER BY approximate_votes DESC LIMIT 9")
	if err != nil {
		return resp.Err("Error while getting certified servers", err)
	}
	listIndex.Certified, err = processRow(d.Context, certRows)
	if err != nil {
		return resp.Err("Error while processing certified servers", err)
	}

	// Premium Servers
	premRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE state = 'public' AND premium = true ORDER BY approximate_votes  DESC LIMIT 9")
	if err != nil {
		return resp.Err("Error while getting premium servers", err)
	}
	listIndex.Premium, err = processRow(d.Context, premRows)
	if err != nil {
		return resp.Err("Error while processing premium servers", err)
	}

	// Most Viewed Servers
	mostViewedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		return resp.Err("Error while getting most viewed servers", err)
	}
	listIndex.MostViewed, err = processRow(d.Context, mostViewedRows)
	if err != nil {
		return resp.Err("Error while processing most viewed servers", err)
	}

	// Recently Added Servers
	recentlyAddedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE state = 'public' AND type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		return resp.Err("Error while getting recently added servers", err)
	}
	listIndex.RecentlyAdded, err = processRow(d.Context, recentlyAddedRows)
	if err != nil {
		return resp.Err("Error while processing recently added servers", err)
	}

	// Top Voted Servers
	topVotedRows, err := state.Pool.Query(d.Context, "SELECT "+indexServersCols+" FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') ORDER BY approximate_votes  DESC LIMIT 9")
	if err != nil {
		return resp.Err("Error while getting top voted servers", err)
	}
	listIndex.TopVoted, err = processRow(d.Context, topVotedRows)
	if err != nil {
		return resp.Err("Error while processing top voted servers", err)
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
		if (servers[i].Type != "approved" && servers[i].Type != "certified") || servers[i].State != "public" {
			return nil, fmt.Errorf("internal error: servers %s has invalid type %s or state %s", servers[i].ServerID, servers[i].Type, servers[i].State)
		}
	}

	// Resolve all servers concurrently, since each server's resolution is independent
	if err := assets.ResolveIndexServers(ctx, servers); err != nil {
		return nil, err
	}

	return servers, nil
}
