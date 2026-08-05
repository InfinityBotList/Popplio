package bot

import (
	"context"
	"errors"
	"strings"

	"popplio/perms"
	"popplio/teams"

	"github.com/disgoorg/snowflake/v2"
)

func checkForPermission(ctx context.Context, guildID snowflake.ID, userID string, perm perms.Perm) error {
	set, err := teams.GetEntityPerms(ctx, userID, "server", guildID.String())

	if err != nil {
		if strings.Contains(err.Error(), "server not found") {
			return errors.New("This server is not on Omniplex! Run `/setup` to enlist it!")
		}

		return err
	}

	if !set.Has(perm) {
		return errors.New("You do not have permission to perform this operation! You must be a member of this server's team with the " + perms.Entity.Label(perm) + " permission.")
	}

	return nil
}
