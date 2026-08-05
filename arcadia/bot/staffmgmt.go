package bot

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// This file is the staff-facing half of permission management: everything
// management needs to see and change who can do what, without opening the panel.
//
// A staff role is a `staff_positions` row bound to a Discord role in the staff
// server. Membership follows Discord: the resync task reads the staff server's
// role assignments and writes them into `staff_members.positions`, so handing
// someone a Discord role is how they get the role's permissions. These commands
// therefore never grant a role to a person — that is Discord's job — they manage
// what a role *means*, and can force a resync when Discord has moved on.

// staffRole is one row of staff_positions.
type staffRole struct {
	ID     string
	Name   string
	RoleID string
	Index  int32
	Perms  []perms.Perm
}

// Mention renders the role as its Discord role mention, which is what makes the
// binding obvious in chat.
func (r staffRole) Mention() string {
	return fmt.Sprintf("<@&%s>", r.RoleID)
}

func registerStaffRoleCommands() {
	register(cmdStaffRoles(), cmdStaffPerms(), cmdPermissions())
}

// managerContext is the caller: what they can do, and how senior they are.
type managerContext struct {
	perms perms.Set
	rank  int32
}

func loadManager(c *Ctx) (managerContext, error) {
	return loadManagerFor(c.Context, c.Author.ID.String())
}

// loadManagerFor is loadManager for a caller that is not the one in hand, which
// is what the permission editor has: its session outlives the invocation that
// opened it and only remembers who owns it.
func loadManagerFor(ctx context.Context, userID string) (managerContext, error) {
	grants, err := perms.LoadStaff(ctx, userID)

	if err != nil {
		return managerContext{}, err
	}

	return managerContext{perms: grants.Resolve(), rank: grants.Rank()}, nil
}

// refuseBotTarget stops a permission being granted to a bot account.
//
// The cached flag on the grants answers it for free when dovewing has seen the
// account before; otherwise the authoritative check runs, which is affordable
// here because granting a permission is a rare, deliberate act.
func refuseBotTarget(ctx context.Context, userID string, target perms.StaffGrants) error {
	if target.BotAccount {
		return fmt.Errorf("<@%s> is a bot: %w", userID, perms.ErrBotAccount)
	}

	if err := perms.RejectBotAccount(ctx, userID); err != nil {
		return fmt.Errorf("<@%s>: %w", userID, err)
	}

	return nil
}

// requireRoleManager loads the caller and checks they may manage roles at all.
func requireRoleManager(c *Ctx) (managerContext, error) {
	manager, err := loadManager(c)

	if err != nil {
		return managerContext{}, err
	}

	if !manager.perms.Has(perms.StaffManageStaffRoles) {
		return managerContext{}, fmt.Errorf("You need the %s permission to use this command", perms.Staff.Label(perms.StaffManageStaffRoles))
	}

	return manager, nil
}

// canEditRole refuses changes to a role at or above the caller's own rank, so
// that nobody can edit their way upwards.
func (m managerContext) canEditRole(role staffRole) error {
	if role.Index <= m.rank {
		return fmt.Errorf("**%s** is rank #%d, which is at or above your own (#%d)", role.Name, role.Index, m.rank)
	}

	return nil
}

func listStaffRoles(ctx context.Context) ([]staffRole, error) {
	rows, err := state.Pool.Query(ctx, "SELECT id::text, name, role_id, index, perms FROM staff_positions ORDER BY index")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var out []staffRole

	for rows.Next() {
		var (
			r     staffRole
			perml []string
		)

		if err := rows.Scan(&r.ID, &r.Name, &r.RoleID, &r.Index, &perml); err != nil {
			return nil, err
		}

		r.Perms = perms.ParseStrings(perml)
		out = append(out, r)
	}

	return out, rows.Err()
}

// lookupStaffRole finds a role by name, by Discord role mention or id, or by its
// own id, so that whichever of the three the caller has to hand works.
func lookupStaffRole(ctx context.Context, input string) (staffRole, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return staffRole{}, errors.New("Which staff role?")
	}

	roleID := strings.Trim(input, "<@&>")

	var (
		r     staffRole
		perml []string
	)

	err := state.Pool.QueryRow(ctx, `
		SELECT id::text, name, role_id, index, perms
		FROM staff_positions
		WHERE lower(name) = lower($1) OR role_id = $2 OR id::text = $2
		LIMIT 1`, input, roleID).Scan(&r.ID, &r.Name, &r.RoleID, &r.Index, &perml)

	if errors.Is(err, pgx.ErrNoRows) {
		names, listErr := listStaffRoles(ctx)

		if listErr != nil || len(names) == 0 {
			return staffRole{}, fmt.Errorf("no staff role called %q", input)
		}

		var labels []string

		for _, n := range names {
			labels = append(labels, n.Name)
		}

		return staffRole{}, fmt.Errorf("no staff role called %q. There is: %s", input, strings.Join(labels, ", "))
	}

	if err != nil {
		return staffRole{}, err
	}

	r.Perms = perms.ParseStrings(perml)

	return r, nil
}

func roleHolderCounts(ctx context.Context) (map[string]int, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT sp.id::text, count(sm.user_id)
		FROM staff_positions sp
		LEFT JOIN staff_members sm ON sp.id = ANY(sm.positions)
		GROUP BY sp.id`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	out := map[string]int{}

	for rows.Next() {
		var (
			id    string
			count int
		)

		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}

		out[id] = count
	}

	return out, rows.Err()
}

// resolvePermission turns what someone typed into a declared permission, and
// says what they might have meant when it does not.
func resolvePermission(input string) (perms.Perm, error) {
	input = strings.ToLower(strings.Trim(strings.TrimSpace(input), "` "))

	if input == "" {
		return "", errors.New("Which permission?")
	}

	perm := perms.Perm(input)

	if err := perms.Staff.Validate(perm); err == nil {
		return perm, nil
	}

	matches := perms.Staff.Suggest(input)

	if len(matches) == 1 {
		return matches[0].ID, nil
	}

	if len(matches) > 1 {
		var names []string

		for _, d := range matches {
			names = append(names, string(d.ID))
		}

		sort.Strings(names)

		return "", fmt.Errorf("there is no permission called ``%s``. Did you mean: %s?", input, strings.Join(names, ", "))
	}

	return "", fmt.Errorf("there is no permission called ``%s``. See ``/permissions`` for the full list", input)
}
