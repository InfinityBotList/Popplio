package assets

import (
	"context"
	"fmt"
	"popplio/assetmanager"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// ApplySelfStatus overrides a resolved user's status with the bot's own
// self-reported presence (see Post Bot Stats), when one has been posted.
// dovewing's gateway-cache-derived status is only populated when the bot
// shares a guild with our own Discord client, which most listed bots don't
// — so without this, almost every bot would show as permanently offline.
func ApplySelfStatus(user *dovetypes.PlatformUser, selfStatus string) {
	if selfStatus == "" || user == nil {
		return
	}
	user.Status = dovetypes.PlatformStatus(selfStatus)
}

func ResolveIndexBot(ctx context.Context, bot *types.IndexBot) error {
	// Set the user for each bot
	botUser, err := dovewing.GetUser(ctx, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying for bot user [dovewing]: %w", err)
	}

	bot.User = botUser
	ApplySelfStatus(bot.User, bot.SelfStatus.String)

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
