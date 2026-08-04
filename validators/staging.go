package validators

import (
	"context"
	"fmt"
	"popplio/config"

	"popplio/perms"
)

// For staging and dev, ensure user is in whitelist
//
// This is because staging and dev use test keys
func StagingCheckSensitive(ctx context.Context, userId string) error {
	// This is because staging and dev use test keys
	if config.CurrentEnv != config.CurrentEnvProd {
		sp, err := perms.StaffPerms(ctx, userId)

		if err != nil {
			return fmt.Errorf("failed to get user staff perms: %w", err)
		}

		if !sp.Has(perms.StaffUseStagingKey) {
			return fmt.Errorf("user does not have the %s staff permission", perms.StaffUseStagingKey)
		}
	}

	return nil
}
