package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"popplio/perms"
	"popplio/teams"

	"github.com/infinitybotlist/eureka/uapi"
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
	perm perms.Perm,
) error {
	if _, ok := uapi.State.AuthTypeMap[targetType]; !ok {
		return ErrInvalidTargetType
	}

	// First check if the action can be performed given permission limits
	//
	// This is a common check and should come first before anything else
	permLimits := PermLimits(authData)

	if len(permLimits) > 0 {
		if !perms.Entity.ResolveStrings(permLimits).Has(perm) {
			return fmt.Errorf("%w: %s", ErrMissingPermission, perm)
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
		if !entityPerms.Has(perm) {
			return fmt.Errorf("%w: %s", ErrMissingPermission, perm)
		}

		return nil
	default:
		return ErrCrossEntityNotSupported
	}
}

// Needs builds the NeededPermission function for a route that always requires
// the same permission, which every route does now that permissions no longer
// vary by the type of entity being acted on.
func Needs(p perms.Perm) func(uapi.Route, *http.Request, uapi.AuthData) (*perms.Perm, error) {
	return func(uapi.Route, *http.Request, uapi.AuthData) (*perms.Perm, error) {
		return &p, nil
	}
}
