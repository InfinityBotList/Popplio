package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// Vote moderation: banning an entity from being voted for, and voiding votes
// already cast, for one entity or for every entity of a target type.
//
// Nothing here deletes a vote. Votes are voided in place with a reason and a
// timestamp, so the record of what was reset survives the reset.

func voteBanSet(ctx context.Context, m *types.RPCTargetReason, h Handle, banned bool) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	query := "UPDATE bots SET vote_banned = false WHERE bot_id = $1"
	if banned {
		query = "UPDATE bots SET vote_banned = true WHERE bot_id = $1"
	}

	if _, err := state.Pool.Exec(ctx, query, m.TargetID); err != nil {
		return Success{}, err
	}

	title := "Vote Ban Removed!"
	description := fmt.Sprintf("<@%s> has removed the vote ban on <@%s>", h.UserID, m.TargetID)

	if banned {
		title = "Vote Ban Edit!"
		description = fmt.Sprintf("<@%s> has set the vote ban on <@%s>", h.UserID, m.TargetID)
	}

	err := modLogReason(
		title,
		description,
		"Remember: don't abuse our services!", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func voteReset(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	_, err := state.Pool.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (single entity) reset', voided_at = NOW() WHERE target_type = $1 AND target_id = $2 AND void = FALSE",
		h.TargetType.String(), m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__Entity Vote Reset!__",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Target ID", Value: m.TargetID, Inline: impls.InlineTrue()},
				{Name: "Target Type", Value: h.TargetType.String(), Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Sad life :("),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func voteResetAll(ctx context.Context, m *types.RPCVoteResetAll, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return Success{}, err
	}

	defer tx.Rollback(ctx)

	// Note: unlike VoteReset this has no void = FALSE filter.
	_, err = tx.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (all entities) reset', voided_at = NOW() WHERE target_type = $1 AND immutable = false",
		h.TargetType.String(),
	)

	if err != nil {
		return Success{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__All Entity Votes Reset!__",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Target Type", Value: h.TargetType.String(), Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Sad life :("),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
