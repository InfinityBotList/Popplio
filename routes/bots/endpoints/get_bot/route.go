package get_bot

import (
	"net/http"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	botColsArr = utils.GetCols(types.Bot{})
	botCols    = strings.Join(botColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:  "GET",
		Path:    "/bots/{id}",
		OpId:    "get_bot",
		Summary: "Get Bot",
		Description: `
Gets a bot by id or name

**Some things to note:**

-` + constants.BackTick + constants.BackTick + `external_source` + constants.BackTick + constants.BackTick + ` shows the source of where a bot came from (Metro Reviews etc etr.). If this is set to ` + constants.BackTick + constants.BackTick + `metro` + constants.BackTick + constants.BackTick + `, then ` + constants.BackTick + constants.BackTick + `list_source` + constants.BackTick + constants.BackTick + ` will be set to the metro list ID where it came from` + `
`,
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "bc-"+name).Val()
	if cache != "" {
		d.Resp <- types.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	var bot types.Bot

	var err error

	row, err := state.Pool.Query(d.Context, "SELECT "+botCols+" FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", name)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	err = pgxscan.ScanOne(&bot, row)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	err = utils.ParseBot(d.Context, state.Pool, &bot, state.Discord, state.Redis)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	bot.UniqueClicks = uniqueClicks

	d.Resp <- types.HttpResponse{
		Json:      bot,
		CacheKey:  "bc-" + name,
		CacheTime: time.Minute * 3,
	}
}
