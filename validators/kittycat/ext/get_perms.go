package ext

import (
	"context"
	"fmt"
	"popplio/state"
	"popplio/validators/kittycat/perms"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type sp struct {
	ID    string   `db:"id"`
	Index int32    `db:"index"`
	Perms []string `db:"perms"`
}

func GetUserStaffPerms(ctx context.Context, userId string) (*perms.StaffPermissions, error) {
	var positions []pgtype.UUID
	var permOverrides []string

	err := state.Pool.QueryRow(ctx, "SELECT positions, perm_overrides FROM staff_members WHERE user_id = $1", userId).Scan(&positions, &permOverrides)

	if err != nil {
		return nil, fmt.Errorf("failed to get staff member: %w", err)
	}

	rows, err := state.Pool.Query(ctx, "SELECT id::text, index, perms FROM staff_positions WHERE id = ANY($1)", positions)

	if err != nil {
		return nil, fmt.Errorf("failed to get staff positions: %w", err)
	}

	defer rows.Close()

	posFull, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[sp])

	if err != nil {
		return nil, fmt.Errorf("failed to collect rows: %w", err)
	}

	var sp = perms.StaffPermissions{
		PermOverrides: permOverrides,
		UserPositions: make([]perms.PartialStaffPosition, len(posFull)),
	}
	for _, pos := range posFull {
		sp.UserPositions = append(sp.UserPositions, perms.PartialStaffPosition{
			ID:    pos.ID,
			Perms: pos.Perms,
			Index: pos.Index,
		})
	}

	return &sp, nil
}
