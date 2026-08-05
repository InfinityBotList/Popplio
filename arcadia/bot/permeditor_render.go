package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"popplio/arcadia/impls"
	"popplio/perms"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5"
)

// Drawing the editor: what the target currently holds, and the menus and buttons
// for changing it.
//
// Everything here is a pure function of freshly loaded state, so the menus can
// never show a picture the database has moved on from.

// permTarget is what a session points at, loaded fresh on every render so the
// menus never show a stale picture of the permissions they are editing.
type permTarget struct {
	// roles is every staff role, for the picker; nil when editing a member.
	roles []staffRole
	// role is the staff role being edited, valid only when picked is set.
	role   staffRole
	picked bool
	// grants is the member's own grants when editing a member.
	grants perms.StaffGrants
	// held is exactly what is stored on the target — a role's permissions or a
	// member's direct grants — and never a member's effective set.
	held []perms.Perm
	// holders is how many people a change to this target reaches.
	holders int
}

func loadPermTarget(ctx context.Context, s *permSession) (permTarget, error) {
	if s.editingMember() {
		grants, err := perms.LoadStaff(ctx, s.UserID)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return permTarget{}, fmt.Errorf("<@%s> is not a staff member", s.UserID)
			}

			return permTarget{}, err
		}

		return permTarget{grants: grants, held: grants.Extras, holders: 1}, nil
	}

	roles, err := listStaffRoles(ctx)

	if err != nil {
		return permTarget{}, err
	}

	t := permTarget{roles: roles}

	if s.RoleID == "" {
		return t, nil
	}

	for _, r := range roles {
		if r.ID == s.RoleID {
			t.role, t.picked, t.held = r, true, r.Perms
			break
		}
	}

	if !t.picked {
		return permTarget{}, errRoleGone
	}

	counts, err := roleHolderCounts(ctx)

	if err != nil {
		return permTarget{}, err
	}

	t.holders = counts[t.role.ID]

	return t, nil
}

// renderPermEditor draws the editor for what is currently stored.
func renderPermEditor(ctx context.Context, s *permSession) (discord.MessageCreate, error) {
	manager, err := loadManagerFor(ctx, s.AuthorID)

	if err != nil {
		return discord.MessageCreate{}, err
	}

	target, err := loadPermTarget(ctx, s)

	if err != nil {
		return discord.MessageCreate{}, err
	}

	held := heldMap(target.held)
	editing := target.picked || s.editingMember()

	embed := discord.Embed{Color: impls.ColourGreen}

	switch {
	case s.editingMember():
		embed.Title = "Editing permissions granted directly"
		embed.Description = fmt.Sprintf("<@%s>", s.UserID)
	case target.picked:
		embed.Title = "Editing staff role: " + target.role.Name
		embed.Description = fmt.Sprintf("%s · rank #%d · %d member(s)", target.role.Mention(), target.role.Index, target.holders)
	default:
		embed.Title = "Editing staff roles"
		embed.Description = "Pick the staff role to edit. Only roles below your own rank are listed."
	}

	if s.Status != "" {
		embed.Description += "\n\n" + s.Status
	}

	if editing {
		if s.editingMember() {
			embed.Fields = append(embed.Fields, discord.EmbedField{
				Name:   "Roles",
				Value:  roleSummary(target.grants),
				Inline: impls.InlineFalse(),
			})
		}

		embed.Fields = append(embed.Fields, permissionFields(perms.Staff.NewSet(target.held...))...)

		embed.Footer = &discord.EmbedFooter{
			Text: "Tick to grant, untick to revoke. ⚠️ is a dangerous permission, 🔒 one you cannot manage.",
		}
	}

	var rows []discord.ContainerComponent

	if !s.editingMember() {
		if row, ok := roleSelectRow(target.roles, s, manager); ok {
			rows = append(rows, row)
		} else if !target.picked {
			embed.Description = "There is no staff role below your own rank for you to edit."
		}
	}

	if editing {
		rows = append(rows, discord.NewActionRow(categorySelect(s, held)))

		if menu, ok := permissionSelect(s, held, manager); ok {
			rows = append(rows, discord.NewActionRow(menu))
		}
	}

	rows = append(rows, discord.NewActionRow(editorButtons(s, editing)...))

	return discord.MessageCreate{Embeds: []discord.Embed{embed}, Components: rows}, nil
}

// editorButtons are the actions that are a press rather than a selection.
//
// The bulk ones only appear with a category open, since that is what they act
// on; offering them before then would be offering to do nothing.
func editorButtons(s *permSession, editing bool) []discord.InteractiveComponent {
	var buttons []discord.InteractiveComponent

	if editing && s.Category != "" {
		buttons = append(buttons,
			discord.NewSuccessButton("Grant all", permButtonGrantAll),
			discord.NewDangerButton("Revoke all", permButtonClearAll),
		)
	}

	// Switching roles without closing the editor is what makes copying a
	// category across two roles a handful of clicks.
	if !s.editingMember() && s.RoleID != "" {
		buttons = append(buttons, discord.NewSecondaryButton("Pick another role", permButtonBack))
	}

	return append(buttons, discord.NewSecondaryButton("Close", permButtonClose))
}

// roleSelectRow lists the roles the caller may edit. Roles at or above their own
// rank are left out rather than shown and refused on click.
func roleSelectRow(roles []staffRole, s *permSession, manager managerContext) (discord.ContainerComponent, bool) {
	options := make([]discord.StringSelectMenuOption, 0, len(roles))

	for _, r := range roles {
		if manager.canEditRole(r) != nil {
			continue
		}

		if len(options) == discordSelectLimit {
			break
		}

		options = append(options, discord.NewStringSelectMenuOption(
			truncate(fmt.Sprintf("#%d · %s", r.Index, r.Name), 100), r.ID).
			WithDescription(fmt.Sprintf("%d permission(s)", len(r.Perms))).
			WithDefault(r.ID == s.RoleID))
	}

	if len(options) == 0 {
		return nil, false
	}

	return discord.NewActionRow(discord.NewStringSelectMenu(permSelectRole, "Staff role to edit", options...)), true
}

// categorySelect is the category picker, labelled with how much of each category
// the target already holds so the interesting ones stand out.
func categorySelect(s *permSession, held map[perms.Perm]bool) discord.StringSelectMenuComponent {
	categories := perms.Staff.Categories()

	if len(categories) > discordSelectLimit {
		categories = categories[:discordSelectLimit]
	}

	options := make([]discord.StringSelectMenuOption, 0, len(categories))

	for _, category := range categories {
		defs := perms.Staff.InCategory(category)
		granted := 0

		for _, d := range defs {
			if held[d.ID] {
				granted++
			}
		}

		options = append(options, discord.NewStringSelectMenuOption(truncate(category, 100), category).
			WithDescription(fmt.Sprintf("%d of %d granted", granted, len(defs))).
			WithDefault(category == s.Category))
	}

	return discord.NewStringSelectMenu(permSelectCategory, "Permission category", options...)
}

// permissionSelect is the editor proper: one multi-select of the open category,
// with the permissions the target already holds preselected, so submitting it
// describes the state wanted rather than a change to make.
func permissionSelect(s *permSession, held map[perms.Perm]bool, manager managerContext) (discord.StringSelectMenuComponent, bool) {
	if s.Category == "" {
		return discord.StringSelectMenuComponent{}, false
	}

	defs := perms.Staff.InCategory(s.Category)

	if len(defs) > discordSelectLimit {
		defs = defs[:discordSelectLimit]
	}

	if len(defs) == 0 {
		return discord.StringSelectMenuComponent{}, false
	}

	options := make([]discord.StringSelectMenuOption, 0, len(defs))

	for _, d := range defs {
		label := d.Name

		if d.Dangerous {
			label = "⚠️ " + label
		}

		// A permission the caller does not hold themselves cannot be granted or
		// revoked by them, so it is marked here instead of only failing on submit.
		if !manager.perms.Has(d.ID) {
			label = "🔒 " + label
		}

		options = append(options, discord.NewStringSelectMenuOption(truncate(label, 100), string(d.ID)).
			WithDescription(truncate(d.Description, 100)).
			WithDefault(held[d.ID]))
	}

	menu := discord.NewStringSelectMenu(permSelectPerms, truncate(s.Category+" permissions", 100), options...).
		WithMinValues(0).
		WithMaxValues(len(options))

	return menu, true
}

func roleSummary(grants perms.StaffGrants) string {
	if len(grants.Roles) == 0 {
		return "None"
	}

	names := make([]string, 0, len(grants.Roles))

	for _, r := range grants.Roles {
		names = append(names, fmt.Sprintf("#%d %s", r.Index, r.Name))
	}

	return strings.Join(names, "\n")
}
