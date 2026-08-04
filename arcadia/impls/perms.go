package impls

import (
	"context"

	"popplio/perms"
)

// GetUserPerms is the LIGHT permission path: positions and overrides only, no
// disciplinaries. Every RPC call and most panel operations resolve through here.
//
// The two queries this used to issue are now one; see [perms.LoadStaff]. Its
// error text names the user it failed on and is shown by the panel raw.
func GetUserPerms(ctx context.Context, userID string) (perms.Set, error) {
	return perms.StaffPerms(ctx, userID)
}
