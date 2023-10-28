package validators

import (
	"context"
	"errors"
	"popplio/config"
	"popplio/state"
)

// For staging, ensure user is a hdev or owner
//
// This is because staging uses test keys
func StagingCheckSensitive(ctx context.Context, userId string) error {
	// For staging, ensure user is a hdev or owner
	//
	// This is because staging uses test keys
	if config.CurrentEnv == config.CurrentEnvStaging {
		var hdev bool
		var owner bool

		err := state.Pool.QueryRow(ctx, "SELECT iblhdev, owner FROM users WHERE user_id = $1", userId).Scan(&hdev, &owner)

		if err != nil {
			return errors.New("unable to determine if user is staff")
		}

		if !hdev && !owner {
			return errors.New("user is not a hdev/owner while being in a staging/test environment")
		}
	}

	return nil
}
