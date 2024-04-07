package get_cache_servers

import (
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	cacheServerColsArr = db.GetCols(types.CacheServer{})
	cacheServerCols    = strings.Join(cacheServerColsArr, ",")

	cacheServerBotColsArr = db.GetCols(types.CacheServerBot{})
	cacheServerBotCols    = strings.Join(cacheServerBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Cache Servers",
		Description: "Returns a list of all available cache servers.",
		Resp:        []types.CacheServer{},
		Params: []docs.Parameter{
			{
				Name:        "include",
				Description: "What extra fields to include, comma-seperated.\n`bots` => Include the bots that are on this cache server.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	row, err := state.Pool.Query(d.Context, "SELECT "+cacheServerCols+" FROM cache_servers")

	if err != nil {
		state.Logger.Error("Error while getting cache servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	cacheServers, err := pgx.CollectRows(row, pgx.RowToStructByName[types.CacheServer])

	if err != nil {
		state.Logger.Error("Error while collecting cache servers", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if len(cacheServers) == 0 {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Handle extra includes
	if r.URL.Query().Get("include") != "" {
		includesSplit := strings.Split(r.URL.Query().Get("include"), ",")

		for _, include := range includesSplit {
			switch include {
			case "bots":
				for i, cacheServer := range cacheServers {
					row, err := state.Pool.Query(d.Context, "SELECT "+cacheServerBotCols+" FROM cache_server_bots WHERE guild_id = $1", cacheServer.GuildID)

					if err != nil {
						state.Logger.Error("Error while getting bots for cache server", zap.Error(err), zap.String("cacheServerID", cacheServer.GuildID))
						return uapi.DefaultResponse(http.StatusInternalServerError)
					}

					bots, err := pgx.CollectRows(row, pgx.RowToStructByName[types.CacheServerBot])

					if err != nil {
						state.Logger.Error("Error while collecting bots for cache server", zap.Error(err), zap.String("cacheServerID", cacheServer.GuildID))
						return uapi.DefaultResponse(http.StatusInternalServerError)
					}

					cacheServers[i].Bots = bots
				}
			}
		}
	}

	return uapi.HttpResponse{
		Json: cacheServers,
	}
}
