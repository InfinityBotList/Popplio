package assets

import (
	"context"
	"fmt"
	"popplio/assetmanager"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/infinitybotlist/eureka/dovewing"
)

func ResolveIndexBot(ctx context.Context, bot *types.IndexBot) error {
	// Set the user for each bot
	botUser, err := dovewing.GetUser(ctx, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying for bot user [dovewing]: %w", err)
	}

	bot.User = botUser

	var code string

	err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", bot.VanityRef).Scan(&code)

	if err != nil {
		return fmt.Errorf("error querying vanity table: %w", err)
	}

	bot.Vanity = code
	bot.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeBot, bot.BotID)

	bot.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, bot.BotID, "bot")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}
