package get_all_bots

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

	"github.com/georgysavva/scany/v2/pgxscan"
)

const perPage = 12

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list. Returns a set of paginated ``IndexBot`` objects",
		Resp:        types.AllBots{},
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "filter",
				Description: "Filter bots by name. Slow and limited to only `bot_id` and `username` filter",
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

	filter := r.URL.Query().Get("filter")

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "allbots-"+strconv.FormatUint(pageNum, 10)+"-"+filter).Val()
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

	var rows pgx.Rows

	if filter != "" {
		rows, err = state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (queue_name ILIKE $1 OR bot_id ILIKE $1) ORDER BY created_at DESC LIMIT $2 OFFSET $3", "%"+filter+"%", limit, offset)
	} else {
		rows, err = state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var bots []types.IndexBot

	err = pgxscan.ScanAll(&bots, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Set the user for each bot
	for i, bot := range bots {
		botUser, err := dovewing.GetDiscordUser(d.Context, bot.BotID)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].User = botUser
	}

	var previous string

	// More optimized string concat
	if pageNum > 2 {
		previous = state.Config.Sites.API + "/bots/all?page=" + strconv.FormatUint(pageNum-1, 10)

		if filter != "" {
			previous += "&filter=" + filter
		}
	}

	var count uint64

	if filter != "" {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE (queue_name ILIKE $1 OR bot_id ILIKE $1)", "%"+filter+"%").Scan(&count)
	} else {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&count)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var next string
	if float64(pageNum+1) <= math.Ceil(float64(count)/perPage) {
		next = state.Config.Sites.API + "/bots/all?page=" + strconv.FormatUint(pageNum+1, 10)

		if filter != "" {
			next += "&filter=" + filter
		}
	}

	data := types.AllBots{
		Count:    count,
		Results:  bots,
		PerPage:  perPage,
		Previous: previous,
		Next:     next,
	}

	return uapi.HttpResponse{
		Json:      data,
		CacheKey:  "allbots-" + strconv.FormatUint(pageNum, 10) + "-" + filter,
		CacheTime: 10 * time.Minute,
	}
}
