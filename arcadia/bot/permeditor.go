package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// This file is the interactive half of permission management.
//
// `/staffroles grant` and `/staffroles revoke` move one permission at a time and
// need the caller to know its name; the editor here shows what a role actually
// holds and lets it be changed by ticking boxes in a select menu. The rules are
// exactly the same either way — rank, and the "you cannot hand out power you do
// not have" check in perms.CheckPatch — because both paths end in the same write.
//
// The editor is deliberately stateless about permissions: every render reloads
// the target from the database, so two people editing the same role see each
// other's changes rather than saving a stale picture over them.

const (
	// permEditorTTL is how long an editor stays live. It is longer than the queue
	// browser's window because reading a category and deciding on it takes longer
	// than paging through bots.
	permEditorTTL = 10 * time.Minute

	// discordSelectLimit is Discord's cap on the options in one select menu.
	discordSelectLimit = 25

	permSelectRole     = "perm:role"
	permSelectCategory = "perm:cat"
	permSelectPerms    = "perm:set"
	permButtonGrantAll = "perm:all"
	permButtonClearAll = "perm:none"
	permButtonBack     = "perm:back"
	permButtonClose    = "perm:close"
)

// permSession is one live permission editor, keyed by the message its menus live
// on. Sessions are author-scoped, like the queue browser's.
type permSession struct {
	AuthorID string
	// RoleID is the staff_positions row being edited, empty until one is picked.
	RoleID string
	// UserID is the staff member whose direct grants are being edited. Exactly
	// one of UserID and RoleID is ever set on a session.
	UserID string
	// Category is the permission category currently open in the menu.
	Category string
	// Status is the outcome of the last change, shown above the menus.
	Status    string
	ExpiresAt time.Time
}

// editingMember distinguishes the two things an editor can point at.
func (s *permSession) editingMember() bool {
	return s.UserID != ""
}

// permSessions is keyed by the message id the menus live on. It is guarded by
// sessionsMu, shared with the other component-driven commands.
var permSessions = map[string]*permSession{}

// errRoleGone is the role being edited having been deleted underneath the
// session, which the editor recovers from rather than reports.
var errRoleGone = errors.New("that staff role no longer exists")

// ---------------------------------------------------------------------------
// Entry points
// ---------------------------------------------------------------------------

// runRolesEdit opens the editor on a staff role. The role is optional: without
// one, the editor opens on its role picker.
func runRolesEdit(c *Ctx) error {
	manager, err := requireRoleManager(c)

	if err != nil {
		return err
	}

	s := &permSession{
		AuthorID:  c.Author.ID.String(),
		ExpiresAt: time.Now().Add(permEditorTTL),
	}

	if input := strings.TrimSpace(c.Option("role", 0)); input != "" {
		role, err := lookupStaffRole(c.Context, input)

		if err != nil {
			return err
		}

		if err := manager.canEditRole(role); err != nil {
			return err
		}

		s.RoleID = role.ID
	}

	return openPermEditor(c, s)
}

// runPermsEdit opens the editor on one staff member's direct grants.
func runPermsEdit(c *Ctx) error {
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

	return openPermEditor(c, &permSession{
		AuthorID:  c.Author.ID.String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(permEditorTTL),
	})
}

// openPermEditor posts the editor and remembers it against the message its menus
// arrived on, which is what the component handler looks the session up by.
func openPermEditor(c *Ctx, s *permSession) error {
	msg, err := renderPermEditor(c.Context, s)

	if err != nil {
		return err
	}

	messageID, err := c.SendTracked(msg)

	if err != nil {
		return err
	}

	sessionsMu.Lock()
	permSessions[messageID.String()] = s
	sessionsMu.Unlock()

	return nil
}

// ---------------------------------------------------------------------------
// Interaction handling
// ---------------------------------------------------------------------------

// handlePermEditor drives every component on an editor message.
func handlePermEditor(c *Ctx, e *events.ComponentInteractionCreate, id, messageID string) {
	// The session is copied out, worked on and written back, so no database
	// round trip happens while the shared session lock is held.
	sessionsMu.Lock()
	stored, ok := permSessions[messageID]

	var s permSession

	if ok {
		s = *stored
	}

	sessionsMu.Unlock()

	if !ok {
		// The bot restarted, or the session was cleaned up: the menus are dead,
		// so say so rather than leaving the click unanswered.
		respondEphemeral(e, "This permission editor is no longer active. Run the command again to get a fresh one.")
		return
	}

	if s.AuthorID != e.User().ID.String() {
		respondEphemeral(e, fmt.Sprintf("This permission editor belongs to <@%s>. Run the command yourself to get your own.", s.AuthorID))
		return
	}

	if time.Now().After(s.ExpiresAt) {
		closePermEditor(e, messageID, "This permission editor timed out. Run the command again to carry on.")
		return
	}

	if id == permButtonClose {
		closePermEditor(e, messageID, "Permission editor closed.")
		return
	}

	values := selectValues(e)

	switch id {
	case permSelectRole:
		if len(values) == 0 {
			return
		}

		// The open category is kept when switching roles, which is what makes
		// "give this to that role too" a two-click job.
		s.RoleID, s.Status = values[0], ""
	case permSelectCategory:
		if len(values) == 0 {
			return
		}

		s.Category, s.Status = values[0], ""
	case permSelectPerms:
		if s.Category == "" {
			return
		}

		s.Status = applyPermSelection(c, &s, values)
	case permButtonGrantAll, permButtonClearAll:
		if s.Category == "" {
			return
		}

		selection, err := bulkSelection(c.Context, &s, id == permButtonGrantAll)

		if err != nil {
			s.Status = fmt.Sprintf("Nothing was changed: %s", err)
			break
		}

		s.Status = applyPermSelection(c, &s, selection)
	case permButtonBack:
		s.RoleID, s.Status = "", ""
	default:
		return
	}

	s.ExpiresAt = time.Now().Add(permEditorTTL)

	msg, err := renderPermEditor(c.Context, &s)

	// A role deleted underneath the session drops the editor back to its picker,
	// rather than stranding it on something that is not there any more.
	if errors.Is(err, errRoleGone) {
		s.RoleID = ""
		s.Status = "The staff role you were editing has been deleted. Pick another one."

		msg, err = renderPermEditor(c.Context, &s)
	}

	if err != nil {
		state.Logger.Error("Failed to render the permission editor", zap.Error(err))
		respondEphemeral(e, fmt.Sprintf("There was an error running this command: %s", err))

		return
	}

	sessionsMu.Lock()

	if current, ok := permSessions[messageID]; ok {
		*current = s
	}

	sessionsMu.Unlock()

	err = e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &msg.Embeds,
		Components: &msg.Components,
	})

	if err != nil {
		state.Logger.Error("Failed to update the permission editor", zap.Error(err))
	}
}

// closePermEditor ends a session and strips the menus off its message, so a
// dead editor cannot be clicked again.
func closePermEditor(e *events.ComponentInteractionCreate, messageID, note string) {
	sessionsMu.Lock()
	delete(permSessions, messageID)
	sessionsMu.Unlock()

	var (
		content    = note
		components = []discord.ContainerComponent{}
		embeds     = []discord.Embed{}
	)

	err := e.UpdateMessage(discord.MessageUpdate{
		Content:    &content,
		Embeds:     &embeds,
		Components: &components,
	})

	if err != nil {
		state.Logger.Error("Failed to close the permission editor", zap.Error(err))
	}
}

// applyPermSelection writes the tick boxes of the open category back to the
// target, and returns the line shown above the menus afterwards.
//
// Only the open category is considered: every permission outside it is carried
// through untouched, so editing one category can never quietly drop another.
func applyPermSelection(c *Ctx, s *permSession, selected []string) string {
	manager, err := loadManagerFor(c.Context, s.AuthorID)

	if err != nil {
		return "Nothing was changed: your own permissions could not be read."
	}

	target, err := loadPermTarget(c.Context, s)

	if err != nil {
		return fmt.Sprintf("Nothing was changed: %s", err)
	}

	if err := permEditAllowed(manager, s, target); err != nil {
		return fmt.Sprintf("Nothing was changed: %s", err)
	}

	held := heldMap(target.held)

	chosen := make(map[perms.Perm]bool, len(selected))

	for _, v := range selected {
		chosen[perms.Perm(v)] = true
	}

	var granted, revoked []perms.Perm

	for _, d := range perms.Staff.InCategory(s.Category) {
		switch {
		case chosen[d.ID] && !held[d.ID]:
			granted = append(granted, d.ID)
		case !chosen[d.ID] && held[d.ID]:
			revoked = append(revoked, d.ID)
		}
	}

	if len(granted) == 0 && len(revoked) == 0 {
		return ""
	}

	current := perms.Staff.NewSet(target.held...)
	next := current.With(granted...).Without(revoked...)

	// You cannot hand out, or take away, power you do not have yourself. The
	// whole selection is refused rather than half-applied, so what the menu shows
	// afterwards is always what was actually stored.
	if blocked := perms.Unmanageable(manager.perms, current, next); len(blocked) > 0 {
		return fmt.Sprintf("Nothing was changed: you do not hold %s yourself.", inlineList(blocked))
	}

	if err := writePermTarget(c.Context, s, next); err != nil {
		return fmt.Sprintf("Nothing was changed: %s", err)
	}

	logPermEdit(c, s, target, granted, revoked)

	return permChangeSummary(s, target, granted, revoked)
}

// bulkSelection is what the "Grant all" and "Revoke all" buttons submit: the
// open category with every permission the caller may manage turned on or off.
//
// Permissions the caller cannot manage are left exactly as they are, so a
// category holding one of them still works with one press instead of being
// refused wholesale.
func bulkSelection(ctx context.Context, s *permSession, granting bool) ([]string, error) {
	manager, err := loadManagerFor(ctx, s.AuthorID)

	if err != nil {
		return nil, errors.New("your own permissions could not be read")
	}

	target, err := loadPermTarget(ctx, s)

	if err != nil {
		return nil, err
	}

	return bulkCategorySelection(s.Category, heldMap(target.held), manager, granting), nil
}

// bulkCategorySelection is the rule behind those buttons, kept apart from the
// loading so it can be read and tested on its own.
func bulkCategorySelection(category string, held map[perms.Perm]bool, manager managerContext, granting bool) []string {
	var selected []string

	for _, d := range perms.Staff.InCategory(category) {
		keep := held[d.ID]

		if manager.perms.Has(d.ID) {
			keep = granting
		}

		if keep {
			selected = append(selected, string(d.ID))
		}
	}

	return selected
}

// permEditAllowed re-checks authority at the moment of the write. The command
// checked it when the editor opened, but a session lives for minutes and the
// caller's own roles can change inside that window.
func permEditAllowed(manager managerContext, s *permSession, target permTarget) error {
	if s.editingMember() {
		if !manager.perms.Has(perms.StaffManageStaffMembers) {
			return fmt.Errorf("you need the %s permission", perms.Staff.Label(perms.StaffManageStaffMembers))
		}

		if target.grants.Rank() < manager.rank {
			return fmt.Errorf("<@%s> outranks you", s.UserID)
		}

		// The editor refused to open on a bot; an account that became one since
		// is refused here too, on the cached flag alone so that the write path
		// stays a database-only operation.
		if target.grants.BotAccount {
			return perms.ErrBotAccount
		}

		return nil
	}

	if !manager.perms.Has(perms.StaffManageStaffRoles) {
		return fmt.Errorf("you need the %s permission", perms.Staff.Label(perms.StaffManageStaffRoles))
	}

	return manager.canEditRole(target.role)
}

func writePermTarget(ctx context.Context, s *permSession, next perms.Set) error {
	if s.editingMember() {
		tag, err := state.Pool.Exec(ctx, "UPDATE staff_members SET perm_overrides = $1 WHERE user_id = $2", next.Strings(), s.UserID)

		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return fmt.Errorf("<@%s> is no longer a staff member", s.UserID)
		}

		return nil
	}

	tag, err := state.Pool.Exec(ctx, "UPDATE staff_positions SET perms = $1 WHERE id = $2", next.Strings(), s.RoleID)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return errRoleGone
	}

	return nil
}

func permChangeSummary(s *permSession, target permTarget, granted, revoked []perms.Perm) string {
	var parts []string

	if len(granted) > 0 {
		parts = append(parts, "granted "+inlineList(granted))
	}

	if len(revoked) > 0 {
		parts = append(parts, "revoked "+inlineList(revoked))
	}

	summary := "You " + strings.Join(parts, " and ")

	if s.editingMember() {
		return fmt.Sprintf("%s for <@%s> directly.", summary, s.UserID)
	}

	return fmt.Sprintf("%s. This applies immediately to the %d member(s) with %s.", summary, target.holders, target.role.Mention())
}

func logPermEdit(c *Ctx, s *permSession, target permTarget, granted, revoked []perms.Perm) {
	var parts []string

	if len(granted) > 0 {
		parts = append(parts, "Granted "+inlineList(granted))
	}

	if len(revoked) > 0 {
		parts = append(parts, "revoked "+inlineList(revoked))
	}

	change := strings.Join(parts, " and ")

	if s.editingMember() {
		logRoleChange(c, "Staff member permissions changed",
			fmt.Sprintf("%s for <@%s> directly.", change, s.UserID))

		return
	}

	logRoleChange(c, "Staff role permissions changed",
		fmt.Sprintf("%s on **%s** (<@&%s>).", change, target.role.Name, target.role.RoleID))
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// heldMap reads a stored permission list exactly, without the super permission's
// implication: an editor must show what would be written back, not what the
// holder can do.
func heldMap(list []perms.Perm) map[perms.Perm]bool {
	out := make(map[perms.Perm]bool, len(list))

	for _, p := range list {
		out[p] = true
	}

	return out
}

func inlineList(list []perms.Perm) string {
	out := make([]string, 0, len(list))

	for _, p := range list {
		out = append(out, "``"+string(p)+"``")
	}

	return strings.Join(out, ", ")
}

func selectValues(e *events.ComponentInteractionCreate) []string {
	data, ok := e.Data.(discord.StringSelectMenuInteractionData)

	if !ok {
		return nil
	}

	return data.Values
}

// respondEphemeral answers an interaction with a message only its clicker sees,
// which is how a refusal reaches them without editing the shared message.
func respondEphemeral(e *events.ComponentInteractionCreate, content string) {
	err := e.CreateMessage(discord.MessageCreate{
		Content: content,
		Flags:   discord.MessageFlagEphemeral,
	})

	if err != nil {
		state.Logger.Error("Failed to answer a permission editor interaction", zap.Error(err))
	}
}
