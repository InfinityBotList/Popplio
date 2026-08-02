package impls

import (
	"context"

	"popplio/arcadia/types"
	"popplio/state"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// GetPlatformUser resolves a Discord user into the panel's PlatformUser shape.
//
// INTEGRATION NOTE (§14c): Arcadia carried its own three-tier cache (live disgo
// cache -> internal_user_cache__discord -> Discord HTTP, with an 8h staleness
// window and a detached background refresh). Popplio already runs eureka's
// dovewing against the SAME table with the SAME 8h expiry, plus a Redis hot
// cache in front, so this delegates rather than standing up a second cache.
//
// Two behavioural differences fall out of that and are covered in the
// conformance notes:
//
//  1. Popplio's dovewing maps Discord's "invisible" presence to "offline".
//     Discord only ever reports invisible users as offline to third parties, so
//     this is not observable in practice.
//  2. The background refresh is dovewing's, which is bounded by its own
//     singleflight rather than spawning a detached goroutine per lookup.
func GetPlatformUser(ctx context.Context, userID string) (types.PlatformUser, error) {
	user, err := dovewing.GetUser(ctx, userID, state.DovewingPlatformDiscord)

	if err != nil {
		return types.PlatformUser{}, err
	}

	return fromDovetypes(user), nil
}

func fromDovetypes(u *dovetypes.PlatformUser) types.PlatformUser {
	if u == nil {
		return types.PlatformUser{}
	}

	return types.PlatformUser{
		ID:          u.ID,
		Username:    u.Username,
		Avatar:      u.Avatar,
		DisplayName: u.DisplayName,
		Bot:         u.Bot,
		Status:      normalizeStatus(string(u.Status)),
	}
}

// normalizeStatus reproduces Arcadia's mapping, where anything unrecognised
// becomes "offline".
func normalizeStatus(status string) string {
	switch status {
	case "online", "idle", "dnd", "invisible", "offline":
		return status
	default:
		return "offline"
	}
}
