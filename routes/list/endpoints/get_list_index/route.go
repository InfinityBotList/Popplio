package get_list_index

import (
	"net/http"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

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
		Summary:     "Get List Index",
		Description: "Gets the index of the list. Returns a ``Index`` object",
		Resp:        types.ListIndexBot{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "indexcache").Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	listIndex := types.ListIndexBot{}

	certRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Certified = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.Certified, certRow)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.Certified {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.Discord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.Certified[i].User = botUser
	}

	premRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE premium = true ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.Premium = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.Premium, premRow)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.Premium {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.Discord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.Premium[i].User = botUser
	}

	mostViewedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.MostViewed = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.MostViewed, mostViewedRow)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.MostViewed {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.Discord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.MostViewed[i].User = botUser
	}

	recentlyAddedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.RecentlyAdded = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.RecentlyAdded, recentlyAddedRow)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.RecentlyAdded {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.Discord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.RecentlyAdded[i].User = botUser
	}

	topVotedRow, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY votes DESC LIMIT 9")
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	listIndex.TopVoted = []types.IndexBot{}
	err = pgxscan.ScanAll(&listIndex.TopVoted, topVotedRow)
	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}
	for i, bot := range listIndex.TopVoted {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.Discord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		listIndex.TopVoted[i].User = botUser
	}

	// Packs
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs ORDER BY created_at DESC")

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

	listIndex.Packs = packs

	return uapi.HttpResponse{
		Json:      listIndex,
		CacheKey:  "indexcache",
		CacheTime: 3 * time.Minute,
	}
}
