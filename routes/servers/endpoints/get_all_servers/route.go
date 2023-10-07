package get_all_servers

import (
	"net/http"
	"strconv"
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

const perPage = 12

var (
	indexServerColsArr = db.GetCols(types.IndexServer{})
	indexServerCols    = strings.Join(indexServerColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Servers",
		Description: "Gets all servers on the list. Returns a set of paginated ``IndexServer`` objects",
		Resp:        types.PagedResult[[]types.IndexServer]{},
		RespName:    "PagedResultIndexServer",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "allservers-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	var rows pgx.Rows

	rows, err = state.Pool.Query(d.Context, "SELECT "+indexServerCols+" FROM servers ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Set the user for each server
	for i := range servers {
		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", servers[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		servers[i].Vanity = code
		servers[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeServers, servers[i].ServerID)
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[[]types.IndexServer]{
		Count:   count,
		Results: servers,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json:      data,
		CacheKey:  "allservers-" + strconv.FormatUint(pageNum, 10),
		CacheTime: 10 * time.Minute,
	}
}