package rpc

import (
	"context"
	"errors"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/google/uuid"
)

// Moving a bot to a different owner, gated by transfer_bots.
//
// A bot is owned by either a user or a team and never both, so there are two
// handlers rather than one: each refuses the case the other exists for.

func transferOwnershipUser(ctx context.Context, m *types.RPCBotTransferOwnershipUser, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	var teamOwner *uuid.UUID

	if err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&teamOwner); err != nil {
		return Success{}, err
	}

	if teamOwner != nil {
		return Success{}, errors.New(" is in a team. Please use BotTransferOwnershipTeam")
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET owner = $2 WHERE bot_id = $1", m.TargetID, m.NewOwner); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		" Ownership Force Update!",
		fmt.Sprintf("<@%s> has force-updated the ownership of <@%s> to <@%s>", h.UserID, m.TargetID, m.NewOwner),
		"Contact support if you think this is a mistake", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func transferOwnershipTeam(ctx context.Context, m *types.RPCBotTransferOwnershipTeam, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	teamID, err := uuid.Parse(m.NewTeam)

	if err != nil {
		return Success{}, errors.New("Invalid team ID")
	}

	var teamOwner *uuid.UUID

	if err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&teamOwner); err != nil {
		return Success{}, err
	}

	if teamOwner == nil {
		return Success{}, errors.New(" is not in a team. Please use TransferOwnership")
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET team_owner = $2 WHERE bot_id = $1", m.TargetID, teamID); err != nil {
		return Success{}, err
	}

	err = modLogReason(
		" Ownership Force Update!",
		fmt.Sprintf("<@%s> has force-updated the ownership of <@%s> to team %s", h.UserID, m.TargetID, teamID),
		"Contact support if you think this is a mistake", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
