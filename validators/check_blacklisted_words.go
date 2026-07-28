package validators

import (
	"context"
	"errors"
	"fmt"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// Returns the system for which this word is blacklisted
func GetWordBlacklistSystems(ctx context.Context, word string) ([]string, error) {
	var systems []string

	err := state.Pool.QueryRow(ctx, "SELECT systems FROM blacklisted_words WHERE word = $1", word).Scan(&systems)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get blacklisted word: %w", err)
	}

	return systems, nil
}
