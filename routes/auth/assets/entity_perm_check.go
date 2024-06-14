package assets

import (
	"context"
	"errors"
	"fmt"
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

func AuthEntityPermCheck(
	ctx context.Context,
	authData uapi.AuthData,
	targetType string,
	targetId string,
	perm string,
) error {
	if _, ok := uapi.State.AuthTypeMap[targetType]; !ok {
		return ErrInvalidTargetType
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

	// Get the permissions of the user
	entityPerms, err := teams.GetEntityPerms(ctx, authData.ID, targetType, targetId)

	if err != nil {
		return err
	}

	// Check if the user has the required permission
	neededPerm := perms.Permission{Namespace: targetType, Perm: perm}
	if !perms.HasPerm(entityPerms, neededPerm) {
		return fmt.Errorf("%w: %s", ErrMissingPermission, neededPerm)
	}

	return nil
}
