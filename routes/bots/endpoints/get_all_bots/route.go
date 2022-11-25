package get_all_bots

import (
	"math"
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"
	"time"

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

func Route(d api.RouteData, r *http.Request) {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		d.Resp <- api.DefaultResponse(http.StatusBadRequest)
		return
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "allbots-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		d.Resp <- api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var bots []types.IndexBot

	err = pgxscan.ScanAll(&bots, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	bots, err = utils.ResolveIndexBot(bots)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
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
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
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

	d.Resp <- api.HttpResponse{
		Json:      data,
		CacheKey:  "allbots-" + strconv.FormatUint(pageNum, 10),
		CacheTime: 10 * time.Minute,
	}
}
