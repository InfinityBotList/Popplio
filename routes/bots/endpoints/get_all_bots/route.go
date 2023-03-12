package get_all_bots

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

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
		Description: "Gets all bots on the list. Returns a ``Index`` object",
		Resp:        types.AllBots{},
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

	// Set the user for each bot
	for i, bot := range bots {
		botUser, err := utils.GetDiscordUser(d.Context, bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].User = botUser
	}

	var previous strings.Builder

	// More optimized string concat
	previous.WriteString(state.Config.Sites.API)
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

	next.WriteString(state.Config.Sites.API)
	next.WriteString("/bots/all?page=")
	next.WriteString(strconv.FormatUint(pageNum+1, 10))

	if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
		next.Reset()
	}

	data := types.AllBots{
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
