package panel

import (
	"context"
	"net/http"

	"popplio/state"
)

// The two existence checks the panel's write operations share: one that answers
// the question, and one that turns a missing row straight into a 404.

func countExists(ctx context.Context, query string, args ...any) (bool, error) {
	var count int64

	if err := state.Pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// requireRow is the "does this id exist" guard the shop, tier, disciplinary-type
// and whitelist mutations all run before updating or deleting. Every one of them
// reports the same frozen message, so the check lives in one place.
//
// It returns a response to send when the row is absent, nil when it is present.
func requireRow(ctx context.Context, query, id string) (*response, error) {
	exists, err := countExists(ctx, query, id)

	if err != nil {
		return nil, newError(err)
	}

	if !exists {
		resp := writeText(http.StatusBadRequest, "Entry with same id does not already exist")
		return &resp, nil
	}

	return nil, nil
}
