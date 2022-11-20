package get_all_packs

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
	indexPackColArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols   = strings.Join(indexPackColArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/packs/all",
		OpId:        "get_all_packs",
		Summary:     "Get All Packs",
		Description: "Gets all packs on the list. Returns a ``Index`` object",
		Tags:        []string{api.CurrentTag},
		Resp:        types.AllPacks{},
	})
}

func Route(d api.RouteData, r *http.Request) {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "pca-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		d.Resp <- types.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	packs := []types.IndexBotPack{}

	err = pgxscan.ScanAll(&packs, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	for i := range packs {
		packs[i].Votes, err = utils.ResolvePackVotes(d.Context, packs[i].URL)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
			return
		}
	}

	var previous strings.Builder

	// More optimized string concat
	previous.WriteString(os.Getenv("SITE_URL"))
	previous.WriteString("/packs/all?page=")
	previous.WriteString(strconv.FormatUint(pageNum-1, 10))

	if pageNum-1 < 1 || pageNum == 0 {
		previous.Reset()
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	var next strings.Builder

	next.WriteString(os.Getenv("SITE_URL"))
	next.WriteString("/packs/all?page=")
	next.WriteString(strconv.FormatUint(pageNum+1, 10))

	if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
		next.Reset()
	}

	data := types.AllPacks{
		Count:    count,
		Results:  packs,
		PerPage:  perPage,
		Previous: previous.String(),
		Next:     next.String(),
	}

	d.Resp <- types.HttpResponse{
		Json:      data,
		CacheKey:  "pca-" + strconv.FormatUint(pageNum, 10),
		CacheTime: 2 * time.Minute,
	}
}
