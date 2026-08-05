package rpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// Taking a bot out of the queue and putting it back, gated by review_bots.
//
// These two are the only handlers that write a staff_general_logs row (see
// audit.go): claiming is the one action whose previous state — who held it
// before — cannot be reconstructed afterwards.

func claim(ctx context.Context, m *types.RPCClaim, h Handle) (Success, error) {
	var (
		botType   string
		claimedBy *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy)

	if err != nil {
		return Success{}, err
	}

	if botType != "pending" {
		return Success{}, errors.New("This bot is not pending review")
	}

	// DEAD BRANCH (reproduced): unreachable because the check above already
	// rejected everything that is not "pending". See CONFORMANCE.md.
	if botType == "testbot" {
		return Success{}, errors.New("This bot is a test bot")
	}

	if !m.Force && claimedBy != nil {
		return Success{}, fmt.Errorf("This bot is already claimed by <@%s>", *claimedBy)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET last_claimed = NOW(), claimed_by = $1 WHERE bot_id = $2", h.UserID, m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "claimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Claimed!",
			Description: fmt.Sprintf("<@%s> has claimed <@%s>", h.UserID, m.TargetID),
			Color:       impls.ColourBlurple,
			Fields: []discord.EmbedField{
				{Name: "Force Claim", Value: strconv.FormatBool(m.Force), Inline: impls.InlineFalse()},
			},
			Footer: impls.Footer("This is completely normal, don't worry!"),
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func unclaim(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType   string
		claimedBy *string
		owner     *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &owner)

	if err != nil {
		return Success{}, err
	}

	if botType == "testbot" {
		return Success{}, errors.New("This bot is a test bot")
	}

	if botType != "pending" {
		return Success{}, errors.New("This bot is not pending review")
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if claimedBy == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed", m.TargetID)
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "unclaimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	// No colour is set on this embed upstream.
	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Unclaimed!",
			Description: fmt.Sprintf("<@%s> has unclaimed <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineFalse()},
			},
			Footer: impls.Footer("This is completely normal, don't worry!"),
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
