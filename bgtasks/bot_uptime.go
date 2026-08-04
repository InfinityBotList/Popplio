package bgtasks

import (
	"context"
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// BotUptimeCheck walks every listed bot and records whether it's currently
// online in the main server, via Popplio's own gateway presence cache.
//
// This is deliberately not Infernoplex's job: Infernoplex never requests the
// privileged Presence intent (see infernoplex/src/main.rs), so it has no
// visibility into other bots' online/offline state. Popplio's own client
// does request it (state.go), so this reads straight from its cache — no
// REST calls, no rate limits, no dependency on Arcadia being configured.
//
// A bot not currently in the main server (never invited, or kicked/left)
// simply has no cache entry and counts as a failed check, same as one that's
// present but offline — both mean "not verifiably up right now".
func BotUptimeCheck(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, "SELECT bot_id FROM bots WHERE type = 'approved' OR type = 'certified'")

	if err != nil {
		return fmt.Errorf("querying listed bots: %w", err)
	}

	var botIDs []string

	for rows.Next() {
		var botID string

		if err := rows.Scan(&botID); err != nil {
			rows.Close()
			return fmt.Errorf("scanning bot_id: %w", err)
		}

		botIDs = append(botIDs, botID)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating listed bots: %w", err)
	}

	mainGuild := state.Config.Servers.Main

	for _, botID := range botIDs {
		userID, err := snowflake.Parse(botID)

		if err != nil {
			// Not a valid snowflake at all — a data problem worth knowing
			// about, but not one that should stop every other bot's check.
			state.Logger.Warn("bot_uptime_check: invalid bot_id, skipping", zap.String("botID", botID))
			continue
		}

		online := isOnline(mainGuild, userID)

		_, err = state.Pool.Exec(ctx,
			`UPDATE bots SET
				total_uptime = total_uptime + 1,
				uptime = uptime + CASE WHEN $2 THEN 1 ELSE 0 END,
				uptime_last_checked = NOW()
			WHERE bot_id = $1`,
			botID, online,
		)

		if err != nil {
			return fmt.Errorf("botID=%s: updating uptime: %w", botID, err)
		}
	}

	return nil
}

// isOnline reports whether the given user has a cached non-offline presence
// in the given guild. A missing cache entry (never seen, or invisible) and
// an explicit offline status are both treated as "not online".
func isOnline(guildID, userID snowflake.ID) bool {
	presence, ok := state.Discord.Caches().Presence(guildID, userID)

	if !ok {
		return false
	}

	return presence.Status != discord.OnlineStatusOffline && presence.Status != discord.OnlineStatusInvisible
}
