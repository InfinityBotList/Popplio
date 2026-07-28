package api

import (
	"context"
	"errors"
	"fmt"
	"popplio/teams"

	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
)

var (
	ErrCrossEntityNotSupported     = errors.New("cross entity actions with this auth type are not supported")
	ErrUsersCannotModifyOtherUsers = errors.New("users cannot modify other users")
	ErrMissingPermission           = errors.New("missing permission")
	ErrInvalidTargetType           = errors.New("invalid target type")
)

func AuthzEntityPermissionCheck(
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
	//
	// This is a common check and should come first before anything else
	permLimits := PermLimits(authData)

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

	switch authData.TargetType {
	case TargetTypeUser:
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
	default:
		return ErrCrossEntityNotSupported
	}
}
