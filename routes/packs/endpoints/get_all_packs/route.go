package get_all_packs

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/georgysavva/scany/v2/pgxscan"
)

const perPage = 12

var (
	indexPackColArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols   = strings.Join(indexPackColArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Packs",
		Description: "Gets all packs on the list. Returns a ``Index`` object",
		Resp:        types.PagedResult[types.IndexBotPack]{},
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
	cache := state.Redis.Get(d.Context, "pca-"+strconv.FormatUint(pageNum, 10)).Val()
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

	rows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	packs := []types.IndexBotPack{}

	err = pgxscan.ScanAll(&packs, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range packs {
		packs[i].Votes, err = utils.ResolvePackVotes(d.Context, packs[i].URL)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := utils.CreatePage(utils.CreatePagedResult[types.IndexBotPack]{
		Count:   count,
		Page:    pageNum,
		PerPage: perPage,
		Path:    "/bots/all",
		Results: packs,
	})

	return uapi.HttpResponse{
		Json:      data,
		CacheKey:  "pca-" + strconv.FormatUint(pageNum, 10),
		CacheTime: 1 * time.Minute,
	}
}
