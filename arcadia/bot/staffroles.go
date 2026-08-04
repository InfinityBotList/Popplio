package bot

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/tasks"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
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

// ---------------------------------------------------------------------------
// /staffroles
// ---------------------------------------------------------------------------

func cmdStaffRoles() *Command {
	roleOption := discord.ApplicationCommandOptionString{
		Name:        "role",
		Description: "Staff role, by name or Discord role",
		Required:    true,
	}

	permOption := discord.ApplicationCommandOptionString{
		Name:        "permission",
		Description: "Permission name, e.g. review_bots",
		Required:    true,
	}

	return &Command{
		Name:        "staffroles",
		Category:    "Staff Management",
		Description: "See and manage staff roles and the permissions attached to them",
		Checks:      []Check{staffServer, isStaff},
		Subcommands: []*Command{
			{
				Name:        "list",
				Category:    "Staff Management",
				Description: "List every staff role, most senior first",
				Checks:      []Check{staffServer, isStaff},
				Run:         runRolesList,
			},
			{
				Name:        "show",
				Category:    "Staff Management",
				Description: "Show a staff role's permissions",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{roleOption},
				Run:         runRolesShow,
			},
			{
				Name:        "create",
				Category:    "Staff Management",
				Description: "Create a staff role bound to a Discord role",
				Checks:      []Check{staffServer, isStaff},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "name", Description: "Name for the new staff role", Required: true},
					discord.ApplicationCommandOptionRole{Name: "discord_role", Description: "The staff server role that grants it", Required: true},
				},
				Run: runRolesCreate,
			},
			{
				Name:        "grant",
				Category:    "Staff Management",
				Description: "Give a permission to a staff role",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{roleOption, permOption},
				Run:         runRolesGrant,
			},
			{
				Name:        "revoke",
				Category:    "Staff Management",
				Description: "Take a permission away from a staff role",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{roleOption, permOption},
				Run:         runRolesRevoke,
			},
			{
				Name:        "rename",
				Category:    "Staff Management",
				Description: "Rename a staff role",
				Checks:      []Check{staffServer, isStaff},
				Options: []discord.ApplicationCommandOption{
					roleOption,
					discord.ApplicationCommandOptionString{Name: "name", Description: "The new name", Required: true},
				},
				Run: runRolesRename,
			},
			{
				Name:        "delete",
				Category:    "Staff Management",
				Description: "Delete a staff role",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{roleOption},
				Run:         runRolesDelete,
			},
			{
				Name:        "sync",
				Category:    "Staff Management",
				Description: "Resync staff roles against the staff server's Discord roles now",
				Checks:      []Check{staffServer, isStaff},
				Run:         runRolesSync,
			},
		},
		Run: func(c *Ctx) error {
			return c.Say("Use one of: ``list``, ``show``, ``create``, ``grant``, ``revoke``, ``rename``, ``delete``, ``sync``.")
		},
	}
}

func runRolesList(c *Ctx) error {
	if err := requirePerm(c, perms.StaffViewStaff); err != nil {
		return err
	}

	roles, err := listStaffRoles(c.Context)

	if err != nil {
		return err
	}

	if len(roles) == 0 {
		return c.Say("There are no staff roles yet. Create one with ``/staffroles create``.")
	}

	holders, err := roleHolderCounts(c.Context)

	if err != nil {
		return err
	}

	var sb strings.Builder

	for _, r := range roles {
		fmt.Fprintf(&sb, "**#%d · %s** %s\n%d permission(s), %d member(s)\n\n",
			r.Index, r.Name, r.Mention(), len(r.Perms), holders[r.ID])
	}

	return c.Send(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Staff roles",
			Description: sb.String(),
			Footer: &discord.EmbedFooter{
				Text: "Lower number = more senior. Membership follows the Discord role.",
			},
			Color: impls.ColourGreen,
		}},
	})
}

func runRolesShow(c *Ctx) error {
	if err := requirePerm(c, perms.StaffViewStaff); err != nil {
		return err
	}

	role, err := lookupStaffRole(c.Context, c.Option("role", 0))

	if err != nil {
		return err
	}

	holders, err := roleHolderCounts(c.Context)

	if err != nil {
		return err
	}

	fields := []discord.EmbedField{
		{Name: "Discord role", Value: role.Mention(), Inline: impls.InlineTrue()},
		{Name: "Rank", Value: fmt.Sprintf("#%d", role.Index), Inline: impls.InlineTrue()},
		{Name: "Members", Value: strconv.Itoa(holders[role.ID]), Inline: impls.InlineTrue()},
	}

	fields = append(fields, permissionFields(perms.Staff.NewSet(role.Perms...))...)

	return c.Send(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:  "Staff role: " + role.Name,
			Fields: fields,
			Footer: &discord.EmbedFooter{
				Text: "Everyone with the Discord role has these permissions.",
			},
			Color: impls.ColourGreen,
		}},
	})
}

func runRolesCreate(c *Ctx) error {
	manager, err := requireRoleManager(c)

	if err != nil {
		return err
	}

	name := strings.TrimSpace(c.Option("name", 0))

	if name == "" {
		return errors.New("A name is required")
	}

	discordRole := c.Option("discord_role", 1)

	roleID, err := snowflake.Parse(strings.Trim(discordRole, "<@&>"))

	if err != nil {
		return fmt.Errorf("%q is not a Discord role", discordRole)
	}

	// The binding is what makes the role real: without a role in the staff
	// server, nobody can ever hold it.
	if _, ok := dclient.Get().Caches().Role(state.Config.Servers.Staff, roleID); !ok {
		return errors.New("That role does not exist in the staff server")
	}

	tx, err := state.Pool.Begin(c.Context)

	if err != nil {
		return err
	}

	defer tx.Rollback(c.Context)

	var taken bool

	err = tx.QueryRow(c.Context,
		"SELECT EXISTS(SELECT 1 FROM staff_positions WHERE name = $1 OR role_id = $2)",
		name, roleID.String()).Scan(&taken)

	if err != nil {
		return err
	}

	if taken {
		return errors.New("A staff role already exists with that name or Discord role")
	}

	// New roles go to the bottom of the hierarchy. Moving them up is a separate,
	// deliberate act through the panel, which handles the reshuffle.
	var index int32

	if err := tx.QueryRow(c.Context, "SELECT COALESCE(MAX(index), 0) + 1 FROM staff_positions").Scan(&index); err != nil {
		return err
	}

	if index <= manager.rank {
		return fmt.Errorf("you cannot create a role at rank #%d, which is at or above your own (#%d)", index, manager.rank)
	}

	_, err = tx.Exec(c.Context,
		"INSERT INTO staff_positions (name, role_id, perms, index) VALUES ($1, $2, '{}', $3)",
		name, roleID.String(), index)

	if err != nil {
		return err
	}

	if err := tx.Commit(c.Context); err != nil {
		return err
	}

	logRoleChange(c, "Staff role created",
		fmt.Sprintf("**%s** (<@&%s>) was created at rank #%d with no permissions.", name, roleID, index))

	return c.Say(fmt.Sprintf(
		"Created staff role **%s** at rank #%d, bound to <@&%s>. Give it permissions with ``/staffroles grant %s <permission>``.",
		name, index, roleID, name))
}

func runRolesGrant(c *Ctx) error {
	return editRolePerms(c, true)
}

func runRolesRevoke(c *Ctx) error {
	return editRolePerms(c, false)
}

// editRolePerms is the shared body of grant and revoke: the two differ only in
// which direction the permission moves.
func editRolePerms(c *Ctx, granting bool) error {
	manager, err := requireRoleManager(c)

	if err != nil {
		return err
	}

	role, err := lookupStaffRole(c.Context, c.Option("role", 0))

	if err != nil {
		return err
	}

	if err := manager.canEditRole(role); err != nil {
		return err
	}

	perm, err := resolvePermission(c.Option("permission", 1))

	if err != nil {
		return err
	}

	current := perms.Staff.NewSet(role.Perms...)
	next := current.With(perm)

	if !granting {
		next = current.Without(perm)
	}

	if current.Equal(next) {
		if granting {
			return fmt.Errorf("**%s** already has %s", role.Name, perms.Staff.Label(perm))
		}

		return fmt.Errorf("**%s** does not have %s", role.Name, perms.Staff.Label(perm))
	}

	// You cannot hand out, or take away, power you do not have yourself.
	if err := perms.CheckPatch(manager.perms, current, next); err != nil {
		return fmt.Errorf("you do not hold %s yourself, so you cannot change it on a role", perms.Staff.Label(perm))
	}

	_, err = state.Pool.Exec(c.Context, "UPDATE staff_positions SET perms = $1 WHERE id = $2", next.Strings(), role.ID)

	if err != nil {
		return err
	}

	verb, preposition := "Granted", "to"

	if !granting {
		verb, preposition = "Revoked", "from"
	}

	logRoleChange(c, "Staff role permissions changed",
		fmt.Sprintf("%s ``%s`` %s **%s** (<@&%s>).", verb, perm, preposition, role.Name, role.RoleID))

	holders, err := roleHolderCounts(c.Context)

	if err != nil {
		return err
	}

	return c.Say(fmt.Sprintf("%s **%s** (``%s``) %s **%s**. This applies immediately to the %d member(s) with %s.",
		verb, perms.Staff.Label(perm), perm, preposition, role.Name, holders[role.ID], role.Mention()))
}

func runRolesRename(c *Ctx) error {
	manager, err := requireRoleManager(c)

	if err != nil {
		return err
	}

	role, err := lookupStaffRole(c.Context, c.Option("role", 0))

	if err != nil {
		return err
	}

	if err := manager.canEditRole(role); err != nil {
		return err
	}

	name := strings.TrimSpace(c.Option("name", 1))

	if name == "" {
		return errors.New("A name is required")
	}

	_, err = state.Pool.Exec(c.Context, "UPDATE staff_positions SET name = $1 WHERE id = $2", name, role.ID)

	if err != nil {
		return err
	}

	logRoleChange(c, "Staff role renamed", fmt.Sprintf("**%s** is now **%s** (<@&%s>).", role.Name, name, role.RoleID))

	return c.Say(fmt.Sprintf("Renamed **%s** to **%s**.", role.Name, name))
}

func runRolesDelete(c *Ctx) error {
	manager, err := requireRoleManager(c)

	if err != nil {
		return err
	}

	role, err := lookupStaffRole(c.Context, c.Option("role", 0))

	if err != nil {
		return err
	}

	if err := manager.canEditRole(role); err != nil {
		return err
	}

	// Deleting a role takes every one of its permissions away from everyone who
	// holds it, so it needs the same authority as revoking them one by one.
	current := perms.Staff.NewSet(role.Perms...)

	if err := perms.CheckPatch(manager.perms, current, perms.Staff.NewSet()); err != nil {
		return fmt.Errorf("you cannot delete **%s**: %w", role.Name, err)
	}

	holders, err := roleHolderCounts(c.Context)

	if err != nil {
		return err
	}

	if _, err := state.Pool.Exec(c.Context, "DELETE FROM staff_positions WHERE id = $1", role.ID); err != nil {
		return err
	}

	logRoleChange(c, "Staff role deleted",
		fmt.Sprintf("**%s** (<@&%s>) was deleted, taking %d permission(s) from %d member(s).",
			role.Name, role.RoleID, len(role.Perms), holders[role.ID]))

	return c.Say(fmt.Sprintf(
		"Deleted **%s**. Its %d member(s) have lost its %d permission(s); the Discord role <@&%s> still exists and now grants nothing.",
		role.Name, holders[role.ID], len(role.Perms), role.RoleID))
}

func runRolesSync(c *Ctx) error {
	if err := requirePerm(c, perms.StaffManageStaffMembers); err != nil {
		return err
	}

	if err := c.Defer(); err != nil {
		return err
	}

	if err := tasks.StaffResync(c.Context); err != nil {
		return fmt.Errorf("resync failed: %w", err)
	}

	return c.Say("Resynced staff roles against the staff server. Anyone whose Discord roles changed now has the matching permissions.")
}

// ---------------------------------------------------------------------------
// /staffperms
// ---------------------------------------------------------------------------

func cmdStaffPerms() *Command {
	userOption := discord.ApplicationCommandOptionUser{
		Name:        "user",
		Description: "The staff member",
		Required:    true,
	}

	permOption := discord.ApplicationCommandOptionString{
		Name:        "permission",
		Description: "Permission name, e.g. review_bots",
		Required:    true,
	}

	return &Command{
		Name:        "staffperms",
		Category:    "Staff Management",
		Description: "See and manage what a staff member can do",
		Checks:      []Check{staffServer, isStaff},
		Subcommands: []*Command{
			{
				Name:        "show",
				Category:    "Staff Management",
				Description: "Show a staff member's roles and permissions",
				Checks:      []Check{staffServer, isStaff},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionUser{Name: "user", Description: "The staff member (defaults to you)"},
				},
				Run: runPermsShow,
			},
			{
				Name:        "check",
				Category:    "Staff Management",
				Description: "Check whether a staff member has a permission, and where it comes from",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{userOption, permOption},
				Run:         runPermsCheck,
			},
			{
				Name:        "grant",
				Category:    "Staff Management",
				Description: "Give a permission to one staff member, on top of their roles",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{userOption, permOption},
				Run:         func(c *Ctx) error { return editMemberExtras(c, true) },
			},
			{
				Name:        "revoke",
				Category:    "Staff Management",
				Description: "Take away a permission granted to one staff member directly",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{userOption, permOption},
				Run:         func(c *Ctx) error { return editMemberExtras(c, false) },
			},
		},
		Run: func(c *Ctx) error {
			return c.Say("Use one of: ``show``, ``check``, ``grant``, ``revoke``.")
		},
	}
}

func runPermsShow(c *Ctx) error {
	userID := c.Option("user", 0)

	if userID == "" {
		userID = c.Author.ID.String()
	}

	// Reading your own permissions needs nothing; reading someone else's does.
	if userID != c.Author.ID.String() {
		if err := requirePerm(c, perms.StaffViewStaff); err != nil {
			return err
		}
	}

	grants, err := perms.LoadStaff(c.Context, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("<@%s> is not a staff member", userID)
		}

		return err
	}

	roleNames := make([]string, 0, len(grants.Roles))

	for _, r := range grants.Roles {
		roleNames = append(roleNames, fmt.Sprintf("#%d %s", r.Index, r.Name))
	}

	if len(roleNames) == 0 {
		roleNames = append(roleNames, "None")
	}

	fields := []discord.EmbedField{
		{Name: "Roles", Value: strings.Join(roleNames, "\n"), Inline: impls.InlineFalse()},
	}

	if len(grants.Extras) > 0 {
		fields = append(fields, discord.EmbedField{
			Name:   "Granted directly",
			Value:  codeList(grants.Extras),
			Inline: impls.InlineFalse(),
		})
	}

	fields = append(fields, permissionFields(grants.Resolve())...)

	return c.Send(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Staff permissions",
			Description: fmt.Sprintf("<@%s>", userID),
			Fields:      fields,
			Color:       impls.ColourGreen,
		}},
	})
}

func runPermsCheck(c *Ctx) error {
	if err := requirePerm(c, perms.StaffViewStaff); err != nil {
		return err
	}

	userID := c.Option("user", 0)

	grants, err := perms.LoadStaff(c.Context, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("<@%s> is not a staff member", userID)
		}

		return err
	}

	perm, err := resolvePermission(c.Option("permission", 1))

	if err != nil {
		return err
	}

	resolved := grants.Resolve()

	if !resolved.Has(perm) {
		return c.Say(fmt.Sprintf("No — <@%s> does not have **%s** (``%s``).", userID, perms.Staff.Label(perm), perm))
	}

	// Saying where it comes from is the point of the command: it is the
	// difference between "revoke it" and "revoke it from four roles".
	var sources []string

	if resolved.IsSuper() && perm != perms.StaffAdministrator {
		sources = append(sources, "Administrator, which implies everything")
	}

	for _, r := range grants.Roles {
		for _, p := range r.Perms {
			if p == perm {
				sources = append(sources, fmt.Sprintf("the **%s** role", r.Name))
				break
			}
		}
	}

	for _, p := range grants.Extras {
		if p == perm {
			sources = append(sources, "a direct grant on them")
			break
		}
	}

	return c.Say(fmt.Sprintf("Yes — <@%s> has **%s** (``%s``), from %s.",
		userID, perms.Staff.Label(perm), perm, strings.Join(sources, " and ")))
}

// editMemberExtras grants or revokes a permission on one person, leaving their
// roles alone. It is the escape hatch for "this one person needs this one
// thing"; anything broader belongs on a role.
func editMemberExtras(c *Ctx, granting bool) error {
	manager, err := loadManager(c)

	if err != nil {
		return err
	}

	if !manager.perms.Has(perms.StaffManageStaffMembers) {
		return fmt.Errorf("You need the %s permission to use this command", perms.Staff.Label(perms.StaffManageStaffMembers))
	}

	userID := c.Option("user", 0)

	target, err := perms.LoadStaff(c.Context, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("<@%s> is not a staff member", userID)
		}

		return err
	}

	if target.Rank() < manager.rank {
		return fmt.Errorf("<@%s> outranks you", userID)
	}

	perm, err := resolvePermission(c.Option("permission", 1))

	if err != nil {
		return err
	}

	current := perms.Staff.NewSet(target.Extras...)
	next := current.With(perm)

	if !granting {
		next = current.Without(perm)
	}

	if current.Equal(next) {
		if granting {
			return fmt.Errorf("<@%s> already has ``%s`` granted directly", userID, perm)
		}

		return fmt.Errorf("<@%s> does not have ``%s`` granted directly — check their roles with ``/staffperms check``", userID, perm)
	}

	if err := perms.CheckPatch(manager.perms, current, next); err != nil {
		return fmt.Errorf("you do not hold %s yourself, so you cannot change it on someone else", perms.Staff.Label(perm))
	}

	_, err = state.Pool.Exec(c.Context, "UPDATE staff_members SET perm_overrides = $1 WHERE user_id = $2", next.Strings(), userID)

	if err != nil {
		return err
	}

	verb, preposition := "Granted", "to"

	if !granting {
		verb, preposition = "Revoked", "from"
	}

	logRoleChange(c, "Staff member permissions changed",
		fmt.Sprintf("%s ``%s`` %s <@%s> directly.", verb, perm, preposition, userID))

	return c.Say(fmt.Sprintf("%s **%s** (``%s``) %s <@%s>.", verb, perms.Staff.Label(perm), perm, preposition, userID))
}

// ---------------------------------------------------------------------------
// /permissions
// ---------------------------------------------------------------------------

func cmdPermissions() *Command {
	return &Command{
		Name:        "permissions",
		Category:    "Staff Management",
		Description: "List the staff permissions that exist and what they mean",
		Checks:      []Check{staffServer, isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "search", Description: "Filter by name or category"},
		},
		Run: func(c *Ctx) error {
			search := strings.ToLower(strings.TrimSpace(c.Option("search", 0)))

			var sb strings.Builder

			for _, category := range perms.Staff.Categories() {
				defs := perms.Staff.InCategory(category)
				var lines []string

				for _, d := range defs {
					if search != "" &&
						!strings.Contains(strings.ToLower(category), search) &&
						!strings.Contains(string(d.ID), search) &&
						!strings.Contains(strings.ToLower(d.Name), search) {
						continue
					}

					marker := ""

					if d.Dangerous {
						marker = " ⚠️"
					}

					lines = append(lines, fmt.Sprintf("``%s``%s — %s", d.ID, marker, d.Description))
				}

				if len(lines) == 0 {
					continue
				}

				fmt.Fprintf(&sb, "\n__**%s**__\n%s\n", category, strings.Join(lines, "\n"))
			}

			if sb.Len() == 0 {
				return fmt.Errorf("no permission matches %q", search)
			}

			// Discord caps a message at 2000 characters and the full catalogue is
			// longer, so the unfiltered listing is split across messages.
			for _, chunk := range chunkMessage(sb.String(), 1900) {
				if err := c.Say(chunk); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// managerContext is the caller: what they can do, and how senior they are.
type managerContext struct {
	perms perms.Set
	rank  int32
}

func loadManager(c *Ctx) (managerContext, error) {
	grants, err := perms.LoadStaff(c.Context, c.Author.ID.String())

	if err != nil {
		return managerContext{}, err
	}

	return managerContext{perms: grants.Resolve(), rank: grants.Rank()}, nil
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

// permissionFields renders a permission set as one embed field per category,
// which is the only way a 30-permission set stays readable.
func permissionFields(set perms.Set) []discord.EmbedField {
	if set.IsEmpty() {
		return []discord.EmbedField{{Name: "Permissions", Value: "None", Inline: impls.InlineFalse()}}
	}

	if set.IsSuper() {
		return []discord.EmbedField{{
			Name:   "Permissions",
			Value:  "**Administrator** — everything",
			Inline: impls.InlineFalse(),
		}}
	}

	var fields []discord.EmbedField

	for _, category := range perms.Staff.Categories() {
		var held []perms.Perm

		for _, d := range perms.Staff.InCategory(category) {
			if set.Has(d.ID) {
				held = append(held, d.ID)
			}
		}

		if len(held) == 0 {
			continue
		}

		fields = append(fields, discord.EmbedField{
			Name:   category,
			Value:  codeList(held),
			Inline: impls.InlineTrue(),
		})
	}

	// Permissions owned by another service still belong to the holder and are
	// shown rather than quietly hidden.
	if undeclared := set.Undeclared(); len(undeclared) > 0 {
		fields = append(fields, discord.EmbedField{
			Name:   "Other services",
			Value:  codeList(undeclared),
			Inline: impls.InlineFalse(),
		})
	}

	return fields
}

func codeList(list []perms.Perm) string {
	out := make([]string, 0, len(list))

	for _, p := range list {
		out = append(out, "``"+string(p)+"``")
	}

	return strings.Join(out, "\n")
}

// chunkMessage splits text on line boundaries to fit Discord's message limit,
// preserving the text exactly across the chunks.
func chunkMessage(s string, limit int) []string {
	var (
		out     []string
		current []string
		size    int
	)

	for _, line := range strings.Split(s, "\n") {
		if size+len(line)+1 > limit && len(current) > 0 {
			out = append(out, strings.Join(current, "\n"))
			current, size = nil, 0
		}

		current = append(current, line)
		size += len(line) + 1
	}

	if len(current) > 0 {
		out = append(out, strings.Join(current, "\n"))
	}

	return out
}

// logRoleChange writes to the staff log channel, the same place the resync task
// reports to. A permission change nobody can see afterwards is not accountable.
func logRoleChange(c *Ctx, title, description string) {
	err := impls.SendChannel(state.Config.Channels.StaffLogs, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       title,
			Description: description,
			Fields: []discord.EmbedField{
				{Name: "By", Value: fmt.Sprintf("<@%s>", c.Author.ID), Inline: impls.InlineTrue()},
			},
			Color: impls.ColourGreen,
		}},
	})

	if err != nil {
		state.Logger.Error("Failed to write staff log for a permission change: " + err.Error())
	}
}
