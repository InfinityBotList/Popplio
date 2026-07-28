package get_all_servers

import (
	"net/http"
	"strconv"
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
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid page number",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	var rows pgx.Rows

	rows, err = state.Pool.Query(d.Context, "SELECT "+indexServerCols+" FROM servers WHERE (type = 'approved' OR type = 'certified') AND state = 'public' ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Failed to query servers [db query]", zap.Error(err), zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		state.Logger.Error("Failed to query servers [collect]", zap.Error(err), zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Resolve all servers
	for i := range servers {
		err := assets.ResolveIndexServer(d.Context, &servers[i])

		if err != nil {
			state.Logger.Error("Error resolving indexserver", zap.Error(err), zap.String("serverID", servers[i].ServerID))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "An error occurred while resolving index server: " + err.Error() + " serverID: " + servers[i].ServerID},
			}
		}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers").Scan(&count)

	if err != nil {
		state.Logger.Error("Failed to query servers [db count]", zap.Error(err), zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[[]types.IndexServer]{
		Count:   count,
		Results: servers,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
