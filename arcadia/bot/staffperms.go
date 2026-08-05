package bot

import (
	"errors"
	"fmt"
	"strings"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5"
)

// /staffperms: what one person can do, and where it comes from.
//
// Direct grants are the escape hatch for "this one person needs this one
// thing"; anything broader belongs on a role, which is why these handlers only
// ever touch staff_members.perm_overrides and never a role.
//
// /permissions lives here too: it answers "what permissions exist at all",
// which is the question people ask right before using one of the above.

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
		// The interactive one goes first here too; see cmdStaffRoles.
		Subcommands: []*Command{
			{
				Name:        "edit",
				Category:    "Staff Management",
				Description: "Grant and revoke a member's own permissions with menus and buttons",
				Checks:      []Check{staffServer, isStaff},
				Options:     []discord.ApplicationCommandOption{userOption},
				Run:         runPermsEdit,
			},
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
			return c.Say("Use one of: ``show``, ``check``, ``edit``, ``grant``, ``revoke``. ``edit`` is the interactive one.")
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

	if err := refuseBotTarget(c.Context, userID, target); err != nil {
		return err
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

	return c.Ok(fmt.Sprintf("%s **%s** (``%s``) %s <@%s>.", verb, perms.Staff.Label(perm), perm, preposition, userID))
}

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
