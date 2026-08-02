package impls

import (
	"context"
	"fmt"

	"popplio/state"

	perms "github.com/infinitybotlist/kittycat/go"
	"github.com/jackc/pgx/v5"
)

// GetUserPerms is the LIGHT permission path: positions and overrides only, no
// disciplinaries. Every RPC call and most panel operations resolve through here.
//
// Both query errors surface with the same wrapper text as the Rust source,
// because the panel shows the message raw.
func GetUserPerms(ctx context.Context, userID string) (perms.StaffPermissions, error) {
	var positions []string
	var permOverrides []string

	err := state.Pool.QueryRow(ctx, "SELECT positions, perm_overrides FROM staff_members WHERE user_id = $1", userID).Scan(&positions, &permOverrides)

	if err != nil {
		return perms.StaffPermissions{}, fmt.Errorf("Error while getting staff perms of user %s: %s", userID, err)
	}

	userPositions, err := partialPositions(ctx, positions)

	if err != nil {
		return perms.StaffPermissions{}, fmt.Errorf("Error while getting staff perms of user %s: %s", userID, err)
	}

	return perms.StaffPermissions{
		UserPositions: userPositions,
		PermOverrides: perms.PFSS(permOverrides),
	}, nil
}

type partialPositionRow struct {
	ID    string   `db:"id"`
	Index int32    `db:"index"`
	Perms []string `db:"perms"`
}

// partialPositions loads the id/index/perms triples kittycat needs to resolve a
// permission set. Position ids are hyphenated UUID strings.
func partialPositions(ctx context.Context, positions []string) ([]perms.PartialStaffPosition, error) {
	rows, err := state.Pool.Query(ctx, "SELECT id, index, perms FROM staff_positions WHERE id = ANY($1)", positions)

	if err != nil {
		return nil, err
	}

	collected, err := pgx.CollectRows(rows, pgx.RowToStructByName[partialPositionRow])

	if err != nil {
		return nil, err
	}

	out := make([]perms.PartialStaffPosition, 0, len(collected))
	for _, p := range collected {
		out = append(out, perms.PartialStaffPosition{
			ID:    p.ID,
			Index: p.Index,
			Perms: perms.PFSS(p.Perms),
		})
	}

	return out, nil
}

// PermStrings renders a resolved permission set back to strings, which is what
// the wire types carry.
func PermStrings(p []perms.Permission) []string {
	out := make([]string, 0, len(p))
	for _, perm := range p {
		out = append(out, perm.String())
	}

	return out
}
