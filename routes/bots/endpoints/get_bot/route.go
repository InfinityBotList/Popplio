package get_bot

import (
	"crypto/sha256"
	"fmt"
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
	return &docs.Doc{
		Summary: "Get Bot",
		Description: `
Gets a bot by id or name

**Some things to note:**

-` + constants.BackTick + constants.BackTick + `external_source` + constants.BackTick + constants.BackTick + ` shows the source of where a bot came from (Metro Reviews etc.). If this is set to ` + constants.BackTick + constants.BackTick + `metro` + constants.BackTick + constants.BackTick + `, then ` + constants.BackTick + constants.BackTick + `list_source` + constants.BackTick + constants.BackTick + ` will be set to the metro list ID where it came from` + `
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
	}
}

func updateClicks(r *http.Request, name string) {
	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return
	}

	if id == "" {
		return
	}

	// Get IP from request and hash it
	hashedIp := fmt.Sprintf("%x", sha256.Sum256([]byte(r.RemoteAddr)))

	// Create transaction
	tx, err := state.Pool.Begin(state.Context)

	if err != nil {
		state.Logger.Error(err)
		return
	}

	defer tx.Rollback(state.Context)

	// Check if the IP has already clicked the bot by checking the unique_clicks row
	var hasClicked bool

	err = tx.QueryRow(state.Context, "SELECT $1 = ANY(unique_clicks) FROM bots WHERE bot_id = $2", hashedIp, id).Scan(&hasClicked)

	if err != nil {
		state.Logger.Error("Error checking", err)
		return
	}

	if !hasClicked {
		// If not, add it to the array
		state.Logger.Info("Adding click for " + id)
		_, err = tx.Exec(state.Context, "UPDATE bots SET unique_clicks = array_append(unique_clicks, $1) WHERE bot_id = $2", hashedIp, id)

		if err != nil {
			state.Logger.Error("Error adding:", err)
			return
		}
	}

	// Commit transaction
	err = tx.Commit(state.Context)

	if err != nil {
		state.Logger.Error(err)
		return
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "bc-"+id).Val()
	if cache != "" {
		if d.IsClient {
			go updateClicks(r, name)
		}

		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var bot types.Bot

	row, err := state.Pool.Query(d.Context, "SELECT "+botCols+" FROM bots WHERE bot_id = $1", id)

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

	ownerUser, err := utils.GetDiscordUser(d.Context, bot.Owner)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.SubPeriodParsed = types.NewInterval(bot.SubPeriod)

	bot.MainOwner = ownerUser

	botUser, err := utils.GetDiscordUser(d.Context, bot.BotID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.User = botUser

	bot.ResolvedAdditionalOwners = []*types.DiscordUser{}

	for _, owner := range bot.AdditionalOwners {
		ownerUser, err := utils.GetDiscordUser(d.Context, owner)

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

	if d.IsClient {
		go updateClicks(r, name)
	}

	return api.HttpResponse{
		Json:      bot,
		CacheKey:  "bc-" + name,
		CacheTime: time.Minute * 3,
	}
}
