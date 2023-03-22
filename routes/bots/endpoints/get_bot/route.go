package get_bot

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

// A bot is a Discord bot that is on the infinitybotlist.

var (
	botColsArr = utils.GetCols(types.Bot{})
	botCols    = strings.Join(botColsArr, ",")

	teamColsArr = utils.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Get Bot",
		Description: `
Gets a bot by id or name

**Some things to note:**`,
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
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error("Resolve Error", err)
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

	if bot.Owner.Valid {
		ownerUser, err := dovewing.GetDiscordUser(d.Context, bot.Owner.String)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusNotFound)
		}

		bot.MainOwner = ownerUser
	} else {
		var team = types.Team{}

		teamBotsRows, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", bot.TeamOwnerID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		err = pgxscan.ScanOne(&team, teamBotsRows)
		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		// Next handle members
		var members = []types.TeamMember{}

		rows, err := state.Pool.Query(d.Context, "SELECT user_id, perms, created_at FROM team_members WHERE team_id = $1", bot.TeamOwnerID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		defer rows.Close()

		for rows.Next() {
			var userId string
			var perms []teams.TeamPermission
			var createdAt time.Time

			err = rows.Scan(&userId, &perms, &createdAt)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}

			user, err := dovewing.GetDiscordUser(d.Context, userId)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}

			members = append(members, types.TeamMember{
				User:      user,
				Perms:     teams.NewPermissionManager(perms).Perms(),
				CreatedAt: createdAt,
			})
		}

		team.Members = members

		// Gets the bots of the team so we can add it to UserBots
		bots, err := utils.ResolveTeamBots(d.Context, team.ID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		team.UserBots = bots

		bot.TeamOwner = &team
	}

	bot.PremiumPeriodLengthParsed = types.NewInterval(bot.PremiumPeriodLength)

	botUser, err := dovewing.GetDiscordUser(d.Context, bot.BotID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot.User = botUser

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
