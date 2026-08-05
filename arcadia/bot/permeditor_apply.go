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
	"go.uber.org/zap"
)

// What a click does: the component handler, and the write it ends in.
//
// Authority is re-checked here rather than trusted from when the editor opened,
// because a session lives for ten minutes and the caller's own roles can change
// inside that window.

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
		content    = ""
		components = []discord.ContainerComponent{}
		embeds     = []discord.Embed{{Description: note, Color: impls.ColourBlurple}}
	)

	// The content is blanked as well as the components: the editor's own message
	// never had any, but an update that only sets the embed would leave whatever
	// was there before sitting above it.
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
