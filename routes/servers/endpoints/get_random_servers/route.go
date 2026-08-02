// Package get_random_servers implements GET /servers/@random — "Get Random
// Servers".
//
// Returns a list of servers from the database in random order
package get_random_servers

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexServerColsArr = db.GetCols(types.IndexServer{})
	indexServerCols    = strings.Join(indexServerColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Random Servers",
		Description: "Returns a list of servers from the database in random order",
		Resp: types.RandomServers{
			Servers: []types.IndexServer{},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexServerCols+" FROM servers WHERE (type = 'approved' OR type = 'certified') AND state = 'public' ORDER BY RANDOM() LIMIT 3")

	if err != nil {
		return resp.Err("Failed to query servers [db query]", err)
	}

	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		return resp.Err("Failed to query servers [db collect]", err)
	}

	for i := range servers {
		err := assets.ResolveIndexServer(d.Context, &servers[i])

		if err != nil {
			return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving index server."+" serverID: "+servers[i].ServerID, err, zap.String("serverID", servers[i].ServerID))
		}
	}

	return uapi.HttpResponse{
		Json: types.RandomServers{
			Servers: servers,
		},
	}
}
