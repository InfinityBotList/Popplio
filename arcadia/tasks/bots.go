package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type unclaimNotification struct {
	BotID       string
	ClaimedBy   string
	LastClaimed time.Time
}

// AutoUnclaim releases bots that have been claimed for over an hour without a
// verdict, then announces each release.
func AutoUnclaim(ctx context.Context) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		"SELECT bot_id, claimed_by, last_claimed FROM bots WHERE claimed_by IS NOT NULL AND NOW() - last_claimed > INTERVAL '1 hour' FOR UPDATE")

	if err != nil {
		return fmt.Errorf("Error while checking for claimed bots: %s", err)
	}

	type claimedBot struct {
		BotID       string     `db:"bot_id"`
		ClaimedBy   *string    `db:"claimed_by"`
		LastClaimed *time.Time `db:"last_claimed"`
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[claimedBot])

	if err != nil {
		return fmt.Errorf("Error while checking for claimed bots: %s", err)
	}

	var notifications []unclaimNotification

	for _, bot := range bots {
		state.Logger.Info("Unclaiming bot", zap.String("botID", bot.BotID))

		if _, err := tx.Exec(ctx, "UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE bot_id = $1", bot.BotID); err != nil {
			return fmt.Errorf("Error while unclaiming bot %s: %s", bot.BotID, err)
		}

		// Only bots with a known reviewer and claim time get announced.
		if bot.ClaimedBy == nil || bot.LastClaimed == nil {
			continue
		}

		notifications = append(notifications, unclaimNotification{
			BotID:       bot.BotID,
			ClaimedBy:   *bot.ClaimedBy,
			LastClaimed: *bot.LastClaimed,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Error while committing transaction: %s", err)
	}

	for _, n := range notifications {
		err := impls.SendChannel(state.Config.Channels.TestingLounge, discord.MessageCreate{
			Content: fmt.Sprintf("<@%s>", n.ClaimedBy),
			Embeds: []discord.Embed{{
				Title: "Auto-Unclaimed Bot",
				Description: fmt.Sprintf(
					"Bot <@%s> was auto-unclaimed (was previously claimed by <@%s> due to it being claimed for over one hour without being approved or denied).\nThis bot was last claimed <t:%d:R>.",
					n.BotID, n.ClaimedBy, n.LastClaimed.Unix()),
				Color: impls.ColourRed,
			}},
		})

		if err != nil {
			return fmt.Errorf("Error while sending message in #lounge: %s", err)
		}

		owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, n.BotID)

		if err != nil {
			return err
		}

		err = impls.SendModLog(discord.MessageCreate{
			Content: owners.MentionUsers(),
			Embeds: []discord.Embed{{
				Title: "Bot Unclaimed!",
				Description: fmt.Sprintf(`
<@%s> has been unclaimed as it was not being actively reviewed.

Don't worry, this is normal, could just be our staff looking more into your bots functionality!

For more information, you can contact the current reviewer <@%s>

*This bot was claimed <t:%d:R>. This is a automated message letting you know about whats going on...*
                            `, n.BotID, n.ClaimedBy, n.LastClaimed.Unix()),
				Footer: impls.Footer("This is completely normal, don't worry!"),
			}},
		})

		if err != nil {
			return fmt.Errorf("Error while sending message in #mod-logs: %s", err)
		}
	}

	return nil
}

// DeletedBots removes bots whose Discord application no longer exists.
func DeletedBots(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, "SELECT bot_id FROM bots")

	if err != nil {
		return fmt.Errorf("Error while fetching all bots: %s", err)
	}

	botIDs, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return fmt.Errorf("Error while fetching all bots: %s", err)
	}

	for _, botID := range botIDs {
		var username string

		err := state.Pool.QueryRow(ctx, "SELECT username FROM internal_user_cache__discord WHERE id = $1", botID).Scan(&username)

		if err != nil {
			state.Logger.Warn("Bot is not in internal_user_cache__discord, forcing indexing of bot", zap.String("botID", botID))

			// Ask Popplio to index the user, then skip this round.
			url := fmt.Sprintf("%s/platform/user/%s?platform=discord", state.Config.Sites.API.Parse(), botID)

			resp, err := httpGet(ctx, url)

			if err != nil {
				state.Logger.Error("Failed to fetch bot from Popplio", zap.String("botID", botID), zap.Error(err))
				continue
			}

			resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				state.Logger.Error("Failed to fetch bot from Popplio", zap.String("botID", botID), zap.Int("status", resp.StatusCode))
			}

			continue
		}

		if !strings.HasPrefix(username, "Deleted User") && !strings.HasPrefix(username, "deleted_user") {
			continue
		}

		state.Logger.Info("Bot is potentially deleted, checking with Discord API", zap.String("botID", botID))

		url := fmt.Sprintf("%s/api/v10/applications/%s/rpc", state.Config.Meta.PopplioProxy, botID)

		resp, err := httpGet(ctx, url)

		if err != nil {
			return fmt.Errorf("Error while fetching RPC endpoint for bot %s: %s", botID, err)
		}

		status := resp.StatusCode
		resp.Body.Close()

		if status >= 200 && status <= 299 {
			// Bot still exists.
			continue
		}

		state.Logger.Info("Bot is deleted from Discord, removing from database", zap.String("botID", botID))

		owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, botID)

		if err != nil {
			return err
		}

		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return fmt.Errorf("Error creating transaction: %s", err)
		}

		if _, err := tx.Exec(ctx, "DELETE FROM bots WHERE bot_id = $1", botID); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("Error while deleting bot %s from database: %s", botID, err)
		}

		err = impls.SendModLog(discord.MessageCreate{
			Content: owners.MentionUsers(),
			Embeds: []discord.Embed{{
				Title:       "Bot Deleted From Discord!",
				URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend.Parse(), botID),
				Description: fmt.Sprintf("`%s` has been deleted from Discord, and so will be removed from list!", botID),
				Fields: []discord.EmbedField{
					{Name: "Bot", Value: botID, Inline: impls.InlineTrue()},
				},
				Footer: impls.Footer("If this is a mistake, please contact support!"),
				Color:  impls.ColourGreen,
			}},
		})

		if err != nil {
			tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

// PremiumRemove strips premium from bots that lost approval or whose
// subscription expired.
func PremiumRemove(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, `
        SELECT bot_id, start_premium_period, premium_period_length, type FROM bots
		WHERE (
			premium = true
			AND (
				(type != 'approved' AND type != 'certified')
				OR (start_premium_period + premium_period_length) < NOW()
			)
		)
        `)

	if err != nil {
		return fmt.Errorf("Error while checking for expired premium bots: %s", err)
	}

	type premiumBot struct {
		BotID               string        `db:"bot_id"`
		StartPremiumPeriod  *time.Time    `db:"start_premium_period"`
		PremiumPeriodLength time.Duration `db:"premium_period_length"`
		Type                string        `db:"type"`
	}

	botRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[premiumBot])

	if err != nil {
		return fmt.Errorf("Error while checking for expired premium bots: %s", err)
	}

	for _, bot := range botRows {
		user, err := impls.GetPlatformUser(ctx, bot.BotID)

		if err != nil {
			return err
		}

		state.Logger.Info("Removing premium from bot", zap.String("botID", bot.BotID))

		if _, err := state.Pool.Exec(ctx, "UPDATE bots SET premium = false WHERE bot_id = $1", bot.BotID); err != nil {
			return fmt.Errorf("Error while removing premium from bot %s: %s", bot.BotID, err)
		}

		owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, bot.BotID)

		if err != nil {
			return err
		}

		var msg string

		if bot.Type != "approved" && bot.Type != "certified" {
			msg = fmt.Sprintf(
				"<@%s> (%s) by %s has been removed from the premium list because it is not/no longer approved or certified.",
				bot.BotID, user.Username, owners.MentionUsers())
		} else {
			msg = fmt.Sprintf(
				"<@%s> (%s) by %s has been removed from the premium list as their subscription has expired.",
				bot.BotID, user.Username, owners.MentionUsers())
		}

		if err := impls.SendModLog(discord.MessageCreate{Content: msg}); err != nil {
			return err
		}
	}

	return nil
}

// japiReqsMade and japiLastRefresh implement the self-imposed budget of 1800
// requests per rolling hour.
var (
	japiReqsMade    atomic.Int64
	japiLastRefresh atomic.Int64
)

const japiHourlyBudget = 1800

type japiData struct {
	Cached bool `json:"cached"`
	Data   struct {
		Message     *string `json:"message"`
		Application *struct {
			ID          string   `json:"id"`
			BotPublic   bool     `json:"bot_public"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		} `json:"application"`
		Bot *struct {
			ID                    string   `json:"id"`
			ApproximateGuildCount *int32   `json:"approximate_guild_count"`
			Username              *string  `json:"username"`
			AvatarURL             *string  `json:"avatarURL"`
			AvatarHash            *string  `json:"avatarHash"`
			PublicFlagsArray      []string `json:"public_flags_array"`
		} `json:"bot"`
	} `json:"data"`
}

// JapiUpdater refreshes guild counts for up to ten stale approved/certified bots
// per run.
//
// BUG FIXED (flagged): upstream computes the hour-reset check as
// `LAST_REFRESH - now >= 3600` on unsigned integers, which underflows on every
// call after the first and makes the budget reset erratically. The correct
// comparison is used here.
//
// config.japi.key exists but upstream never sends it, and neither does this port
// - see CONFORMANCE.md.
func JapiUpdater(ctx context.Context) error {
	now := time.Now().Unix()

	if now-japiLastRefresh.Load() >= 3600 {
		japiReqsMade.Store(0)
		japiLastRefresh.Store(now)
	}

	rows, err := state.Pool.Query(ctx,
		"SELECT bot_id FROM bots WHERE (type = 'approved' OR type = 'certified') AND (last_stats_post IS NULL OR NOW() - last_stats_post > INTERVAL '3 days') AND (last_japi_update IS NULL OR NOW() - last_japi_update > INTERVAL '3 days') ORDER BY RANDOM() LIMIT 10")

	if err != nil {
		return err
	}

	botIDs, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return err
	}

	for _, botID := range botIDs {
		if japiReqsMade.Add(1) > japiHourlyBudget {
			return errors.New("Internal error: JAPI rate limit hit")
		}

		resp, err := httpGet(ctx, fmt.Sprintf("https://japi.rest/discord/v1/application/%s", botID))

		if err != nil {
			return err
		}

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			resp.Body.Close()
			state.Logger.Error("Failed to fetch bot from JAPI", zap.String("botID", botID))
			continue
		}

		var data japiData

		decodeErr := json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()

		if decodeErr != nil {
			return decodeErr
		}

		if data.Data.Bot != nil {
			_, err = state.Pool.Exec(ctx,
				"UPDATE bots SET last_japi_update = NOW(), servers = $1 WHERE bot_id = $2",
				data.Data.Bot.ApproximateGuildCount, botID)
		} else {
			_, err = state.Pool.Exec(ctx, "UPDATE bots SET last_japi_update = NOW() WHERE bot_id = $1", botID)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
