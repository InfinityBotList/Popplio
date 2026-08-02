// Package get_all_bots implements GET /bots/@all — "Get All Bots".
//
// Gets all bots on the list. Returns a set of paginated `IndexBot` objects
package get_all_bots

import (
	"net/http"
	"popplio/api/resp"
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
		return resp.Err("Error while getting all bots [db fetch]", err)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		return resp.Err("Error while getting all bots [collect]", err)
	}

	// Resolve all bots
	for i := range bots {
		err := assets.ResolveIndexBot(d.Context, &bots[i])

		if err != nil {
			return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot."+" botID: "+bots[i].BotID, err, zap.String("botID", bots[i].BotID))
		}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&count)

	if err != nil {
		return resp.Err("Error while getting bot count", err)
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
