package perms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// staffQuery loads a staff member's extra permissions and every role they hold
// in one round trip, ordered by rank so the most senior role comes first.
const staffQuery = `SELECT
	sm.perm_overrides,
	COALESCE((
		SELECT json_agg(json_build_object('id', sp.id::text, 'name', sp.name, 'index', sp.index, 'perms', sp.perms) ORDER BY sp.index)
		FROM staff_positions sp
		WHERE sp.id = ANY(sm.positions)
	), '[]'::json)
FROM staff_members sm
WHERE sm.user_id = $1`

// StaffGrants is where a staff member's permissions come from: the roles they
// hold, which the Discord resync keeps in step with the staff server, plus the
// extras granted to them individually.
type StaffGrants struct {
	Roles  []Role `json:"roles"`
	Extras []Perm `json:"extras"`

	// ConfigOwner marks an instance owner from the config. See [IsConfigOwner]:
	// they hold everything and outrank everyone, whatever the database says.
	ConfigOwner bool `json:"config_owner"`
}

// Resolve is every permission the member has.
func (g StaffGrants) Resolve() Set {
	set := Staff.Resolve(g.Roles, g.Extras...)

	if g.ConfigOwner {
		return set.With(StaffAdministrator)
	}

	return set
}

// TopRole is the member's most senior role, which is what rank checks compare
// against. The second return is false for a member with no roles at all.
func (g StaffGrants) TopRole() (Role, bool) {
	if len(g.Roles) == 0 {
		return Role{}, false
	}

	top := g.Roles[0]

	for _, r := range g.Roles[1:] {
		if r.Index < top.Index {
			top = r
		}
	}

	return top, true
}

// Rank is the member's seniority: the index of their most senior role, lower
// being more senior. A member with no roles ranks below everyone; an instance
// owner outranks everyone.
func (g StaffGrants) Rank() int32 {
	if g.ConfigOwner {
		return OwnerRank
	}

	top, ok := g.TopRole()

	if !ok {
		return NoRank
	}

	return top.Index
}

// NoRank is the rank of someone who holds no roles. Every real role outranks
// it, so a rank check against it always fails closed.
const NoRank int32 = 1<<31 - 1

type staffRoleJSON struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Index int32    `json:"index"`
	Perms []string `json:"perms"`
}

// LoadStaff returns where a staff member's permissions come from.
//
// Use it when the roles themselves matter — the panel and the staff bot show
// them, and disciplinaries are applied on top. For a plain access check use
// [StaffPerms].
//
// A user with no staff_members row is not staff and the error wraps
// pgx.ErrNoRows.
func LoadStaff(ctx context.Context, userID string) (StaffGrants, error) {
	var (
		extras []string
		roles  []byte
	)

	owner := IsConfigOwner(userID)

	err := state.Pool.QueryRow(ctx, staffQuery, userID).Scan(&extras, &roles)

	if err != nil {
		// An owner is an owner before the resync has ever run, so a missing row
		// is not an obstacle for them.
		if owner && errors.Is(err, pgx.ErrNoRows) {
			return StaffGrants{ConfigOwner: true}, nil
		}

		return StaffGrants{}, fmt.Errorf("error while getting staff perms of user %s: %w", userID, err)
	}

	var rows []staffRoleJSON

	if err := json.Unmarshal(roles, &rows); err != nil {
		return StaffGrants{}, fmt.Errorf("error while getting staff roles of user %s: %w", userID, err)
	}

	g := StaffGrants{
		Roles:       make([]Role, 0, len(rows)),
		Extras:      ParseStrings(extras),
		ConfigOwner: owner,
	}

	for _, row := range rows {
		g.Roles = append(g.Roles, Role{
			ID:    row.ID,
			Name:  row.Name,
			Index: row.Index,
			Perms: ParseStrings(row.Perms),
		})
	}

	return g, nil
}

// StaffPerms resolves a staff member's permissions.
//
// Disciplinaries are not applied here; this is the light path used by RPC calls
// and most panel operations. The panel's own member view applies them on top.
func StaffPerms(ctx context.Context, userID string) (Set, error) {
	g, err := LoadStaff(ctx, userID)

	if err != nil {
		return Set{}, err
	}

	return g.Resolve(), nil
}
