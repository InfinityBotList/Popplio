package tasks

import (
	"fmt"
	"sort"
	"strings"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// What the resync says it did, and the small set operations behind deciding it.
//
// A permission change nobody can see afterwards is not accountable, which is
// why the log is written per member rather than as one summary — but a log that
// fails must never take the resync down with it.

// announceResync posts a resync record to the staff log channel.
//
// A failure here is reported and stepped over rather than returned. The log is
// an account of what the resync did; the resync itself is what keeps staff
// permissions in step with Discord. Returning the error used to abandon the run
// and roll back its transaction, so a staff log channel that had been deleted —
// or, more often, never configured — silently stopped staff permissions syncing
// altogether.
func announceResync(userID string, msg discord.MessageCreate) {
	if err := impls.SendChannel(state.Config.Channels.StaffLogs, msg); err != nil {
		state.Logger.Warn("Could not write the staff log for a resync; the resync itself continued",
			zap.String("userID", userID),
			zap.Error(err),
		)
	}
}

func isOwner(userID snowflake.ID) bool {
	return perms.IsConfigOwner(userID.String())
}

// differs reports whether the two position sets have a non-empty symmetric
// difference.
func differs(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return true
	}

	for k := range a {
		if _, ok := b[k]; !ok {
			return true
		}
	}

	return false
}

// sortedKeys gives the set a stable order so log output and SQL arrays do not
// churn between runs.
func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))

	for k := range set {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

func buildPermissions(posByID map[string]cachedPosition, positionIDs []string, extras []perms.Perm) perms.StaffGrants {
	g := perms.StaffGrants{
		Roles:  make([]perms.Role, 0, len(positionIDs)),
		Extras: extras,
	}

	for _, id := range positionIDs {
		pos, ok := posByID[id]

		if !ok {
			continue
		}

		g.Roles = append(g.Roles, perms.Role{
			ID:    pos.ID,
			Name:  pos.Name,
			Index: pos.Index,
			Perms: perms.ParseStrings(pos.Perms),
		})
	}

	return g
}

// renderPositions formats a position list for the staff-log embed, matching the
// "- “value“" lines upstream emits.
func renderPositions(posByID map[string]cachedPosition, positionIDs []string) string {
	lines := make([]string, 0, len(positionIDs))

	for _, id := range positionIDs {
		if pos, ok := posByID[id]; ok {
			lines = append(lines, fmt.Sprintf("- ``%s``", pos))
		} else {
			lines = append(lines, fmt.Sprintf("- Unknown Position: %s", id))
		}
	}

	if len(lines) == 0 {
		return "None"
	}

	return strings.Join(lines, "\n")
}

func renderPerms(resolved perms.Set) string {
	lines := make([]string, 0, resolved.Len())

	for _, perm := range resolved.All() {
		lines = append(lines, fmt.Sprintf("- ``%s``", perm))
	}

	if len(lines) == 0 {
		return "None"
	}

	return strings.Join(lines, "\n")
}
