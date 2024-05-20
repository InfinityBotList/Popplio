package get_random_servers

import (
	"net/http"
	"popplio/assetmanager"
	"popplio/db"
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
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexServerCols+" FROM servers WHERE (type = 'approved' OR type = 'certified') ORDER BY RANDOM() LIMIT 3")

	if err != nil {
		state.Logger.Error("Failed to query servers [db query]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		state.Logger.Error("Failed to query servers [db collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range servers {
		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", servers[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error("Failed to query vanity [db queryrow]", zap.Error(err), zap.String("server_id", servers[i].ServerID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		servers[i].Vanity = code
		servers[i].Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeServers, servers[i].ServerID)
		servers[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeServers, servers[i].ServerID)
	}

	return uapi.HttpResponse{
		Json: types.RandomServers{
			Servers: servers,
		},
	}
}
