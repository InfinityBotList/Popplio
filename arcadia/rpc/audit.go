package rpc

import (
	"context"
	"encoding/json"

	"popplio/state"
)

// The staff_general_logs row, which is separate from the rpc_logs row Execute
// writes for every call.
//
// rpc_logs records that a method ran and what it was given; this records what
// the world looked like before it did, which is the part that cannot be
// reconstructed afterwards.

// staffGeneralLog writes the claim/unclaim audit row.
func staffGeneralLog(ctx context.Context, userID, action, targetID string, claimedByPrev *string) error {
	data, err := json.Marshal(map[string]any{
		"target_id":       targetID,
		"claimed_by_prev": claimedByPrev,
	})

	if err != nil {
		return err
	}

	_, err = state.Pool.Exec(ctx, "INSERT INTO staff_general_logs (user_id, action, data) VALUES ($1, $2, $3)", userID, action, data)

	return err
}
