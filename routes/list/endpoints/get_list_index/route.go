package get_list_index

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	indexPackColsArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/index",
		OpId:        "get_list_index",
		Summary:     "Get List Index",
		Description: "Gets the index of the list. Returns a ``Index`` object",
		Tags:        []string{api.CurrentTag},
		Resp:        types.ListIndex{},
	})
}

func Route(d api.RouteData, r *http.Request) {
	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "indexcache").Val()
	if cache != "" {
		d.Resp <- api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	listIndex := types.ListIndex{}

	certRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	certDat := []types.IndexBot{}
	err = pgxscan.ScanAll(&certDat, certRow)
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	certDat, err = utils.ResolveIndexBot(certDat)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listIndex.Certified = certDat

	mostViewedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}
	mostViewedDat := []types.IndexBot{}
	err = pgxscan.ScanAll(&mostViewedDat, mostViewedRow)
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	mostViewedDat, err = utils.ResolveIndexBot(mostViewedDat)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listIndex.MostViewed = mostViewedDat

	recentlyAddedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}
	recentlyAddedDat := []types.IndexBot{}
	err = pgxscan.ScanAll(&recentlyAddedDat, recentlyAddedRow)
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	recentlyAddedDat, err = utils.ResolveIndexBot(recentlyAddedDat)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listIndex.RecentlyAdded = recentlyAddedDat

	topVotedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}
	topVotedDat := []types.IndexBot{}
	err = pgxscan.ScanAll(&topVotedDat, topVotedRow)
	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}
	topVotedDat, err = utils.ResolveIndexBot(topVotedDat)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listIndex.TopVoted = topVotedDat

	rows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	packs := []types.IndexBotPack{}

	err = pgxscan.ScanAll(&packs, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	for i := range packs {
		packs[i].Votes, err = utils.ResolvePackVotes(d.Context, packs[i].URL)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
			return
		}
	}

	listIndex.Packs = packs

	d.Resp <- api.HttpResponse{
		Json:      listIndex,
		CacheKey:  "indexcache",
		CacheTime: 15 * time.Minute,
	}
}
