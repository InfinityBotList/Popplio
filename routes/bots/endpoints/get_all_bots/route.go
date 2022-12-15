package get_all_bots

import (
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
)

const perPage = 12

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

type AllBots struct {
	Count    uint64           `json:"count"`
	PerPage  uint64           `json:"per_page"`
	Next     string           `json:"next"`
	Previous string           `json:"previous"`
	Results  []types.IndexBot `json:"bots"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/bots/all",
		OpId:        "get_all_bots",
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list. Returns a ``Index`` object",
		Tags:        []string{api.CurrentTag},
		Resp:        AllBots{},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "allbots-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var bots []types.IndexBot

	err = pgxscan.ScanAll(&bots, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	bots, err = utils.ResolveIndexBot(bots)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var previous strings.Builder

	// More optimized string concat
	previous.WriteString(os.Getenv("SITE_URL"))
	previous.WriteString("/bots/all?page=")
	previous.WriteString(strconv.FormatUint(pageNum-1, 10))

	if pageNum-1 < 1 || pageNum == 0 {
		previous.Reset()
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var next strings.Builder

	next.WriteString(os.Getenv("SITE_URL"))
	next.WriteString("/bots/all?page=")
	next.WriteString(strconv.FormatUint(pageNum+1, 10))

	if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
		next.Reset()
	}

	data := AllBots{
		Count:    count,
		Results:  bots,
		PerPage:  perPage,
		Previous: previous.String(),
		Next:     next.String(),
	}

	return api.HttpResponse{
		Json:      data,
		CacheKey:  "allbots-" + strconv.FormatUint(pageNum, 10),
		CacheTime: 10 * time.Minute,
	}
}
