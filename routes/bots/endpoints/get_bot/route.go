package get_bot

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"popplio/assetmanager"
	"popplio/config"
	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	botColsArr = db.GetCols(types.Bot{})
	botCols    = strings.Join(botColsArr, ",")

	teamColsArr = db.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot",
		Description: "Gets a bot by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID",
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
			{
				Name:        "short",
				Description: "Avoid sending large fields. Currently this is only the long description of the bot",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func handleAnalytics(r *http.Request, id, target string) error {
	switch target {
	case "page":
		// Get IP from request and hash it
		hashedIp := fmt.Sprintf("%x", sha256.Sum256([]byte(r.RemoteAddr)))

		// Create transaction
		tx, err := state.Pool.Begin(state.Context)

		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}

		defer tx.Rollback(state.Context)

		_, err = tx.Exec(state.Context, "UPDATE bots SET clicks = clicks + 1 WHERE bot_id = $1", id)

		if err != nil {
			return fmt.Errorf("error updating clicks count: %w", err)
		}

		// Check if the IP has already clicked the bot by checking the unique_clicks row
		var hasClicked bool

		err = tx.QueryRow(state.Context, "SELECT $1 = ANY(unique_clicks) FROM bots WHERE bot_id = $2", hashedIp, id).Scan(&hasClicked)

		if err != nil {
			return fmt.Errorf("error checking for any unique clicks from this user: %w", err)
		}

		if !hasClicked {
			// If not, add it to the array
			state.Logger.Debug("Adding new unique click for user during handleAnalytics", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("targetType", "bot"))
			_, err = tx.Exec(state.Context, "UPDATE bots SET unique_clicks = array_append(unique_clicks, $1) WHERE bot_id = $2", hashedIp, id)

			if err != nil {
				return fmt.Errorf("error adding new unique click for user: %w", err)
			}
		}

		// Commit transaction
		err = tx.Commit(state.Context)

		if err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
		}
	case "invite":
		// Update clicks
		_, err := state.Pool.Exec(state.Context, "UPDATE bots SET invite_clicks = invite_clicks + 1 WHERE bot_id = $1", id)

		if err != nil {
			return fmt.Errorf("error updating invite clicks: %w", err)
		}
	}

	return nil
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	target := r.URL.Query().Get("target")

	row, err := state.Pool.Query(d.Context, "SELECT "+botCols+" FROM bots WHERE bot_id = $1", id)

	if err != nil {
		state.Logger.Error("Error while getting bot [db fetch]", zap.Error(err), zap.String("id", id), zap.String("target", target))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bot, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Bot])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Error while getting bot [db collect]", zap.Error(err), zap.String("id", id), zap.String("target", target))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if bot.Owner.Valid {
		ownerUser, err := dovewing.GetUser(d.Context, bot.Owner.String, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Error while getting bot owner [dovewing fetch]", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("owner", bot.Owner.String))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bot.MainOwner = ownerUser
	} else {
		row, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", bot.TeamOwnerID)

		if err != nil {
			state.Logger.Error("Error while getting bot team owner [db fetch]", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("teamOwner", assets.EncodeUUID(bot.TeamOwnerID.Bytes)))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

		if err != nil {
			state.Logger.Error("Error while getting bot team owner [collect]", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("teamOwner", assets.EncodeUUID(bot.TeamOwnerID.Bytes)))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto.Entities = &types.TeamEntities{
			Targets: []string{}, // We don't provide any entities right now, may change
		}

		eto.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeTeams, eto.ID)
		eto.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, eto.ID)

		bot.TeamOwner = &eto
	}

	botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Error while getting bot user [dovewing fetch]", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bot.User = botUser

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

	if err != nil {
		state.Logger.Error("Error while getting bot unique clicks [db fetch]", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bot.UniqueClicks = uniqueClicks

	bot.LegacyWebhooks = config.UseLegacyWebhooks(bot.BotID)

	var code string

	err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", bot.VanityRef).Scan(&code)

	if err != nil {
		state.Logger.Error("Error while getting bot vanity code [db fetch]", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bot.Vanity = code
	bot.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeBots, bot.BotID)

	go func() {
		err = handleAnalytics(r, id, target)

		if err != nil {
			state.Logger.Error("Error while handling analytics", zap.Error(err), zap.String("id", id), zap.String("target", target))
		}
	}()

	if r.URL.Query().Get("short") == "true" {
		bot.Long = ""
	}

	return uapi.HttpResponse{
		Json: bot,
	}
}
