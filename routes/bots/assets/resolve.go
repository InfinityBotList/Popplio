// Package assets holds the bot logic shared between endpoints.
//
// It covers resolving a bot into its index representation and refreshing bot
// metadata from Discord, both of which are needed by several endpoints and
// by the team entity resolvers.
package assets

import (
	"context"
	"fmt"
	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

const statsPostFreshness = 24 * time.Hour

func ApplySelfStatus(user *dovetypes.PlatformUser, selfStatus string, servers int, lastStatsPost pgtype.Timestamptz) {
	if user == nil {
		return
	}

	if selfStatus != "" {
		user.Status = dovetypes.PlatformStatus(selfStatus)
		return
	}

	if servers > 0 && lastStatsPost.Valid && time.Since(lastStatsPost.Time) < statsPostFreshness {
		user.Status = dovetypes.PlatformStatusOnline
	}
}

func ResolveIndexBot(ctx context.Context, bot *types.IndexBot) error {
	botUser, err := dovewing.GetUser(ctx, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying for bot user [dovewing]: %w", err)
	}

	bot.User = botUser
	ApplySelfStatus(bot.User, bot.SelfStatus.String, bot.Servers, bot.LastStatsPost)

	var code string

	err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", bot.VanityRef).Scan(&code)

	if err != nil {
		return fmt.Errorf("error querying vanity table: %w", err)
	}

	bot.Vanity = code

	bot.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, bot.BotID, "bot")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}
