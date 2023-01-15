package get_list_index

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	indexPackColsArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Summary:     "Get List Index",
		Description: "Gets the index of the list. Returns a ``Index`` object",
		Resp:        types.ListIndex{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "indexcache").Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	listIndex := types.ListIndex{}

	certRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Certified = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.Certified, certRow)
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.Certified {
		botUser, err := utils.GetDiscordUser(bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.Certified[i].User = botUser
	}

	mostViewedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.MostViewed = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.MostViewed, mostViewedRow)
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.MostViewed {
		botUser, err := utils.GetDiscordUser(bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.MostViewed[i].User = botUser
	}

	recentlyAddedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.RecentlyAdded = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.RecentlyAdded, recentlyAddedRow)
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.RecentlyAdded {
		botUser, err := utils.GetDiscordUser(bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.RecentlyAdded[i].User = botUser
	}

	topVotedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.TopVoted = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.TopVoted, topVotedRow)
	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.TopVoted {
		botUser, err := utils.GetDiscordUser(bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.TopVoted[i].User = botUser
	}

	// Packs
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	packs := []types.IndexBotPack{}

	err = pgxscan.ScanAll(&packs, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range packs {
		packs[i].Votes, err = utils.ResolvePackVotes(d.Context, packs[i].URL)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	listIndex.Packs = packs

	return api.HttpResponse{
		Json:      listIndex,
		CacheKey:  "indexcache",
		CacheTime: 15 * time.Minute,
	}
}
