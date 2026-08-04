// Package teams implements team membership and the permissions attached to
// it.
//
// The permissions themselves are declared in [perms.Entity]; this package
// answers "what may this user do with this entity". Entity ownership in Popplio
// is either a single user or a team, so most authorization decisions on bots,
// servers and packs end up here.
package teams

import (
	"context"
	"errors"
	"fmt"

	"popplio/perms"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// IsValidPerm reports whether a permission name is one a team can actually
// grant.
func IsValidPerm(perm string) bool {
	return perms.Entity.Validate(perms.Perm(perm)) == nil
}

// GetEntityPerms returns what a user may do with an entity.
//
// - For a bot or server the entity's team decides, unless it is owned by a
// single user, in which case that user has everything and nobody else has
// anything.
// - For a team it is the user's membership of that team.
// - A user always has full control over themselves.
//
// The result is a resolved [perms.Set] in the [perms.Entity] domain, ready to
// be queried with Has or handed to [perms.CheckPatch].
func GetEntityPerms(ctx context.Context, userId, targetType, targetId string) (perms.Set, error) {
	var teamId string

	switch targetType {
	case "user":
		// Special case
		if targetId != userId {
			return perms.Set{}, fmt.Errorf("users do not have permissions on other users")
		}

		return perms.Entity.NewSet(perms.EntityOwner), nil
	case "bot":
		var teamOwner pgtype.Text
		var owner pgtype.Text
		err := state.Pool.QueryRow(ctx, "SELECT team_owner, owner FROM bots WHERE bot_id = $1", targetId).Scan(&teamOwner, &owner)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("bot not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding bot: %v", err)
		}

		if owner.Valid {
			// Fast path, no lookup of team membership needed at all
			if owner.String == userId {
				return perms.Entity.NewSet(perms.EntityOwner), nil
			}

			return perms.Set{}, nil
		}

		teamId = teamOwner.String
	case "team":
		teamId = targetId
	case "server":
		var teamOwner pgtype.Text
		err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM servers WHERE server_id = $1", targetId).Scan(&teamOwner)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("server not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding server: %v", err)
		}

		teamId = teamOwner.String
	default:
		return perms.Set{}, fmt.Errorf("invalid target type")
	}

	// Handle teams
	if _, err := uuid.Parse(teamId); err != nil {
		return perms.Set{}, fmt.Errorf("invalid team id")
	}

	// Get the team member from the team
	var teamPerms []string
	err := state.Pool.QueryRow(ctx, "SELECT flags FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&teamPerms)

	if errors.Is(err, pgx.ErrNoRows) {
		return perms.Set{}, nil
	}

	if err != nil {
		return perms.Set{}, fmt.Errorf("error finding team member: %v", err)
	}

	// Teams have no roles, so a member's flags are the whole source
	return perms.Entity.ResolveStrings(teamPerms), nil
}
