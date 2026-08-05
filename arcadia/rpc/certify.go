package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

// Certification, gated by certify_bots.
//
// Certifying is a change of the bot's type rather than a flag of its own, so
// uncertifying returns it to "approved" and never to "pending".

func certifyAdd(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'certified' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		" Force Certified!",
		fmt.Sprintf("<@%s> has force-certified <@%s>", h.UserID, m.TargetID),
		"Neat", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func certifyRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'approved' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		" Uncertified!",
		fmt.Sprintf("<@%s> has uncertified <@%s>", h.UserID, m.TargetID),
		"Uh oh, looks like you've been naughty...", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
