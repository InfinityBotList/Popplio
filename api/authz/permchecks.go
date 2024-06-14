package authz

import (
	"context"
	"errors"
	"fmt"
	"popplio/api"
	"popplio/teams"

	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
)

var (
	ErrOnlyUsersCanPerformCrossEntityActions = errors.New("only users can perform cross entity actions")
	ErrUsersCannotModifyOtherUsers           = errors.New("users cannot modify other users")
	ErrMissingPermission                     = errors.New("missing permission")
	ErrInvalidTargetType                     = errors.New("invalid target type")
)

func EntityPermissionCheck(
	ctx context.Context,
	authData uapi.AuthData,
	targetType string,
	targetId string,
	perm perms.Permission,
) error {
	if _, ok := uapi.State.AuthTypeMap[targetType]; !ok {
		return ErrInvalidTargetType
	}

	// First check if the action can be performed given permission limits
	permLimits := api.PermLimits(authData)

	if len(permLimits) > 0 {
		var resolvedPermLimits = perms.StaffPermissions{
			PermOverrides: perms.PFSS(permLimits),
		}.Resolve()

		if !perms.HasPerm(resolvedPermLimits, perm) {
			return fmt.Errorf("%w: %s", ErrMissingPermission, perm.String())
		}
	}

	// If target type == authData.TargetType and targetId == authData.TargetId, then return true
	if targetType == authData.TargetType && targetId == authData.ID {
		return nil
	}

	// If not authenticated as a user at this point, error as only users can perform cross entity actions
	if authData.TargetType != "user" {
		return ErrOnlyUsersCanPerformCrossEntityActions
	}

	// If the targetType is a user, error as users cannot modify other users
	if targetType == "user" {
		return ErrUsersCannotModifyOtherUsers
	}

	// Get the permissions of the user to ensure that the underlying user can also perform the action
	entityPerms, err := teams.GetEntityPerms(ctx, authData.ID, targetType, targetId)

	if err != nil {
		return err
	}

	// Check if the user has the required permission
	if !perms.HasPerm(entityPerms, perm) {
		return fmt.Errorf("%w: %s", ErrMissingPermission, perm.String())
	}

	return nil
}
