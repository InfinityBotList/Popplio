package assets

import (
	"context"
	"errors"
	"fmt"
	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"strings"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexBotColArr = db.GetCols(types.IndexBot{})
	indexBotCols   = strings.Join(indexBotColArr, ",")
)

func ResolveBotPack(ctx context.Context, pack *types.BotPack) error {
	ownerUser, err := dovewing.GetUser(ctx, pack.Owner, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying dovewing for owner user: %w", err)
	}

	pack.ResolvedOwner = ownerUser

	// Ensure this always marshals as `[]` rather than `null` when the pack
	// has no bots — a nil Go slice serializes to JSON null, which crashes
	// frontend consumers that call .length/.map on it without a null check.
	pack.ResolvedBots = []types.IndexBot{}

	for _, botId := range pack.Bots {
		row, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE bot_id = $1", botId)

		if err != nil {
			state.Logger.Error("Error querying bots table [db fetch]", zap.Error(err), zap.String("bot_id", botId))
			return fmt.Errorf("error querying bots table: %w", err)
		}

		bot, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.IndexBot])

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			return fmt.Errorf("error querying bots table: %w", err)
		}

		// Resolve the bot
		err = assets.ResolveIndexBot(ctx, &bot)

		if err != nil {
			return fmt.Errorf("error occurred while resolving index bot: " + err.Error() + " botID: " + bot.BotID)
		}

		pack.ResolvedBots = append(pack.ResolvedBots, bot)
	}

	pack.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, pack.URL, "pack")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}
