package assets

import (
	"context"
	"popplio/state"
)

// The Review Nest Engine calculates the depths of reviews
func Nest(ctx context.Context, id string) int {
	var depth int

	var reachedRoot bool

	for !reachedRoot {
		var parent string
		err := state.Pool.QueryRow(ctx, "SELECT parent_id FROM reviews WHERE id = $1", id).Scan(&parent)

		if err != nil {
			state.Logger.Error(err)
			return depth
		}

		if parent == "" {
			reachedRoot = true
		} else {
			id = parent
			depth++
		}
	}

	return depth
}
