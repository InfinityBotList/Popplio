package validators

import (
	"context"
	"fmt"
	"popplio/config"
	"popplio/validators/kittycat/ext"
	"popplio/validators/kittycat/perms"
)

// For staging, ensure user is in whitelist
//
// This is because staging uses test keys
func StagingCheckSensitive(ctx context.Context, userId string) error {
	// This is because staging uses test keys
	if config.CurrentEnv == config.CurrentEnvStaging {
		sp, err := ext.GetUserStaffPerms(ctx, userId)

		if err != nil {
			return fmt.Errorf("failed to get user staff perms: %w", err)
		}

		if !perms.HasPerm(sp.Resolve(), perms.Build("popplio_staging", "sensitive")) {
			return fmt.Errorf("user does not have the popplio_staging.sensitive staff permission")
		}
	}

	return nil
}
