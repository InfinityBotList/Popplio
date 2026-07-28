package assets

import (
	"context"
	"errors"
	"fmt"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// The Review Nest Engine calculates the depths of reviews
func Nest(ctx context.Context, id string) (int, error) {
	var depth int

	var reachedRoot bool

	for !reachedRoot {
		var parent pgtype.Text
		err := state.Pool.QueryRow(ctx, "SELECT parent_id FROM reviews WHERE id = $1", id).Scan(&parent)

		if errors.Is(err, pgx.ErrNoRows) {
			return depth, nil
		}

		if err != nil {
			return depth, fmt.Errorf("failed to query parent_id of id %s: %w", id, err)
		}

		if !parent.Valid || parent.String == "" {
			reachedRoot = true
		} else {
			id = parent.String
			depth++
		}
	}

	return depth, nil
}
