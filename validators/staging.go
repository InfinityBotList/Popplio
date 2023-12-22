package validators

import (
	"context"
	"errors"
	"popplio/config"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

// For staging, ensure user is in whitelist
//
// This is because staging uses test keys
func StagingCheckSensitive(ctx context.Context, userId string) error {
	// For staging, ensure user is a hdev or owner
	//
	// This is because staging uses test keys
	if config.CurrentEnv == config.CurrentEnvStaging {
		var positions []pgtype.UUID
		var permOverrides []string

		rec, err := state.Pool.QueryRow(ctx, "SELECT positions, perm_overrides FROM staff_members WHERE user_id = $1", userId)

		if !hdev && !owner {
			return errors.New("user is not a hdev/owner while being in a staging/test environment")
		}
	}

	return nil
}
