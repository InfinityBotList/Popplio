package assets

import (
	"context"
	"errors"
	"fmt"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// The Review Nest Engine calculates the depths of reviews
func Nest(ctx context.Context, id string) (int, error) {
	var depth int

	var reachedRoot bool

	for !reachedRoot {
		var parent string
		err := state.Pool.QueryRow(ctx, "SELECT parent_id FROM reviews WHERE id = $1", id).Scan(&parent)

		if errors.Is(err, pgx.ErrNoRows) {
			return depth, nil
		}

		if err != nil {
			return depth, fmt.Errorf("failed to query parent_id of id %s: %w", id, err)
		}

		if parent == "" {
			reachedRoot = true
		} else {
			id = parent
			depth++
		}
	}

	return depth, nil
}
