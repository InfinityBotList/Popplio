package bot

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"popplio/perms"

	"github.com/jackc/pgx/v5"
)

// The interactive permission editor: its session, and the two commands that
// open one.
//
// `/staffroles grant` and `/staffroles revoke` move one permission at a time and
// need the caller to know its name; the editor shows what a role actually holds
// and lets it be changed by ticking boxes. The rules are the same either way —
// rank, and the "you cannot hand out power you do not have" check in
// perms.CheckPatch — because both paths end in the same write.
//
// The editor is deliberately stateless about permissions: every render reloads
// the target from the database (permeditor_render.go), so two people editing the
// same role see each other's changes rather than saving a stale picture over
// them.

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
