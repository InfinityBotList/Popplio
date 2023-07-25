package get_bot

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"popplio/config"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

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
		Summary:     "Get Bot",
		Description: "Gets a bot by id/vanity",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name: "target",
				Description: `The target page of the request if any. 
				
If target is 'page', then unique clicks will be counted based on a SHA-256 hashed IP

If target is 'invite', then the invite will be counted as a click

Officially recognized targets:

- page -> bot page view
- settings -> bot settings page view
- stats -> bot stats page view
- invite -> bot invite view
- vote -> bot vote page`,
				Required: false,
				In:       "query",
				Schema:   docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func handleAnalytics(r *http.Request, id, target string) {
	switch target {
	case "page":
		// Get IP from request and hash it
		hashedIp := fmt.Sprintf("%x", sha256.Sum256([]byte(r.RemoteAddr)))

		// Create transaction
		tx, err := state.Pool.Begin(state.Context)

		if err != nil {
			state.Logger.Error(err)
			return
		}

		defer tx.Rollback(state.Context)

		_, err = tx.Exec(state.Context, "UPDATE bots SET clicks = clicks + 1")

		if err != nil {
			state.Logger.Error(err)
			return
		}

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
	case "invite":
		// Update clicks
		_, err := state.Pool.Exec(state.Context, "UPDATE bots SET invite_clicks = invite_clicks + 1 WHERE bot_id = $1", id)

		if err != nil {
			state.Logger.Error(err)
		}
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	name := chi.URLParam(r, "id")

	target := r.URL.Query().Get("target")

	// Resolve bot ID
	id, err := utils.ResolveBot(d.Context, name)

	if err != nil {
		state.Logger.Error("Resolve Error", err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "bc-"+id).Val()
	if cache != "" {
		go handleAnalytics(r, id, target)
		return uapi.HttpResponse{
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
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&bot, row)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if utils.IsNone(bot.Banner.String) || !strings.HasPrefix(bot.Banner.String, "https://") {
		bot.Banner.Valid = false
		bot.Banner.String = ""
	}

	if bot.Owner.Valid {
		ownerUser, err := dovewing.GetUser(d.Context, bot.Owner.String, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bot.MainOwner = ownerUser
	} else {
		// Convert pgtype.UUID to string
		team, err := utils.ResolveTeam(d.Context, utils.UUIDString(bot.TeamOwnerID))

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bot.TeamOwner = team
	}

	bot.PremiumPeriodLengthParsed = types.NewInterval(bot.PremiumPeriodLength)

	botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	bot.User = botUser

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	bot.UniqueClicks = uniqueClicks

	bot.LegacyWebhooks = config.UseLegacyWebhooks(bot.BotID)

	go handleAnalytics(r, id, target)

	return uapi.HttpResponse{
		Json:      bot,
		CacheKey:  "bc-" + name,
		CacheTime: time.Minute * 3,
	}
}
