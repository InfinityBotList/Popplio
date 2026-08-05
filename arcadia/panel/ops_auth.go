package panel

import (
	"context"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
)

// Who is calling and what they may do — the opening of nearly every operation
// in this package.

// helloVersion is the protocol version the panel must send.
const helloVersion = 5

// checkAuth wraps the auth check so failures surface the way Error::new does:
// status 500 with the bare message the frontend matches on.
func checkAuth(ctx context.Context, token string) (types.AuthData, error) {
	data, err := impls.CheckAuth(ctx, token)

	if err != nil {
		return types.AuthData{}, newError(err)
	}

	return data, nil
}

func checkAuthInsecure(ctx context.Context, token string) (types.AuthData, error) {
	data, err := impls.CheckAuthInsecure(ctx, token)

	if err != nil {
		return types.AuthData{}, newError(err)
	}

	return data, nil
}

// authorize is the opening of nearly every panel operation: who is calling, and
// what they are allowed to do.
//
// The two are always wanted together, and an operation that took one without
// the other would be one that either cannot say who acted or cannot refuse
// them. Operations needing rank as well as permissions use impls.GetStaffMember
// instead, which is the heavier path.
func authorize(ctx context.Context, token string) (types.AuthData, perms.Set, error) {
	authData, err := checkAuth(ctx, token)

	if err != nil {
		return types.AuthData{}, perms.Set{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return types.AuthData{}, perms.Set{}, err
	}

	return authData, userPerms, nil
}

// resolvedPerms is the light permission path used by most operations.
func resolvedPerms(ctx context.Context, userID string) (perms.Set, error) {
	sp, err := impls.GetUserPerms(ctx, userID)

	if err != nil {
		return perms.Set{}, newError(err)
	}

	return sp, nil
}
