package get_all_bots

import (
	"net/http"
	"strconv"
	"strings"

	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const perPage = 12

var (
	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list. Returns a set of paginated ``IndexBot`` objects",
		Resp:        types.PagedResult[[]types.IndexBot]{},
		RespName:    "PagedResultIndexBot",
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

	limit := perPage
	offset := (pageNum - 1) * perPage

	var rows pgx.Rows

	rows, err = state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error("Error while getting all bots [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error("Error while getting all bots [collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Resolve all bots
	for i := range bots {
		err := assets.ResolveIndexBot(d.Context, &bots[i])

		if err != nil {
			state.Logger.Error("Error resolving indexbot", zap.Error(err), zap.String("botID", bots[i].BotID))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "An error occurred while resolving index bot: " + err.Error() + " botID: " + bots[i].BotID},
			}
		}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&count)

	if err != nil {
		state.Logger.Error("Error while getting bot count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[[]types.IndexBot]{
		Count:   count,
		Results: bots,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
