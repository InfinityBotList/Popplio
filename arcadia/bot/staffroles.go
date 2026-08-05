package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/tasks"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// /staffroles: what a staff role means.
//
// These handlers never grant a role to a person — that is Discord's job, and the
// resync task mirrors it — they change what holding the role gets you. The
// interactive editor behind `edit` is in permeditor.go.

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

	// The editor can open on its own role picker, so naming a role is optional
	// there and required everywhere else.
	optionalRoleOption := roleOption
	optionalRoleOption.Required = false

	return &Command{
		Name:        "staffroles",
		Category:    "Staff Management",
		Description: "See and manage staff roles and the permissions attached to them",
		Checks:      []Check{staffServer, isStaff},
		// Discord lists subcommands in registration order and never lets the
		// parent command be run on its own, so the interactive one goes first:
		// it is what most of this command is for.
		Subcommands: []*Command{
			{
				Name:        "edit",
				Category:    "Staff Management",
				Description: "Grant and revoke a role's permissions with menus and buttons",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{optionalRoleOption},
				Run:         runRolesEdit,
			},
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
			return c.Say("Use one of: ``list``, ``show``, ``edit``, ``create``, ``grant``, ``revoke``, ``rename``, ``delete``, ``sync``. ``edit`` is the interactive one.")
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

	return c.Ok(fmt.Sprintf(
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

	return c.Ok(fmt.Sprintf("%s **%s** (``%s``) %s **%s**. This applies immediately to the %d member(s) with %s.",
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

	return c.Ok(fmt.Sprintf("Renamed **%s** to **%s**.", role.Name, name))
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

	return c.Ok(fmt.Sprintf(
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

	return c.Ok("Resynced staff roles against the staff server. Anyone whose Discord roles changed now has the matching permissions.")
}
