package get_bot

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

// A bot is a Discord bot that is on the infinitybotlist.

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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "bc-"+name).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	// First check count so we can avoid expensive DB calls
	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE "+constants.ResolveBotSQL, name).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	if count > 1 {
		// Delete one of the bots
		_, err := state.Pool.Exec(d.Context, "DELETE FROM bots WHERE "+constants.ResolveBotSQL+" LIMIT 1", name)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	var bot types.Bot

	row, err := state.Pool.Query(d.Context, "SELECT "+botCols+" FROM bots WHERE "+constants.ResolveBotSQL, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&bot, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if utils.IsNone(bot.Banner.String) || !strings.HasPrefix(bot.Banner.String, "https://") {
		bot.Banner.Valid = false
		bot.Banner.String = ""
	}

	if utils.IsNone(bot.Invite.String) || !strings.HasPrefix(bot.Invite.String, "https://") {
		bot.Invite.Valid = false
		bot.Invite.String = ""
	}

	ownerUser, err := utils.GetDiscordUser(bot.Owner)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.SubPeriodParsed = types.NewInterval(bot.SubPeriod)

	bot.MainOwner = ownerUser

	botUser, err := utils.GetDiscordUser(bot.BotID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.User = botUser

	bot.ResolvedAdditionalOwners = []*types.DiscordUser{}

	for _, owner := range bot.AdditionalOwners {
		ownerUser, err := utils.GetDiscordUser(owner)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		bot.ResolvedAdditionalOwners = append(bot.ResolvedAdditionalOwners, ownerUser)
	}

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.UniqueClicks = uniqueClicks

	return api.HttpResponse{
		Json:      bot,
		CacheKey:  "bc-" + name,
		CacheTime: time.Minute * 3,
	}
}
