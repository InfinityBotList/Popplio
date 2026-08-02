package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"sort"
	"strings"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	perms "github.com/infinitybotlist/kittycat/go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// cachedPosition is a staff position with everything the resync needs.
type cachedPosition struct {
	ID                 string
	Name               string
	RoleID             string
	Index              int32
	Perms              []string
	CorrespondingRoles []types.Link
}

// String is the Display impl used in the staff-log embeds.
func (c cachedPosition) String() string {
	return fmt.Sprintf("%s [%s] (<@&%s>)", c.ID, c.Name, c.RoleID)
}

// StaffResync makes staff_members agree with Discord role assignments in the
// STAFF guild and mirrors "corresponding roles" into the other guilds.
//
// This is the highest-risk task in the system: it can delete staff rows and strip
// roles.
func StaffResync(ctx context.Context) error {
	// Step 1: snapshot the staff guild. If it is not cached we abort the whole
	// task rather than acting on partial data.
	if _, ok := state.Discord.Caches().Guild(state.Config.Servers.Staff); !ok {
		return fmt.Errorf("Failed to get staff guild for staff perms resync")
	}

	type guildMember struct {
		UserID snowflake.ID
		Roles  []string
	}

	var staffResync []guildMember

	state.Discord.Caches().MembersForEach(state.Config.Servers.Staff, func(member discord.Member) {
		roles := make([]string, 0, len(member.RoleIDs))

		for _, roleID := range member.RoleIDs {
			roles = append(roles, roleID.String())
		}

		staffResync = append(staffResync, guildMember{UserID: member.User.ID, Roles: roles})
	})

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	// Step 2: load every position into three lookup maps.
	rows, err := tx.Query(ctx, "SELECT id, name, role_id, index, perms, corresponding_roles FROM staff_positions")

	if err != nil {
		return fmt.Errorf("Error while getting staff positions: %v", err)
	}

	type positionDBRow struct {
		ID                 pgtype.UUID `db:"id"`
		Name               string      `db:"name"`
		RoleID             string      `db:"role_id"`
		Index              int32       `db:"index"`
		Perms              []string    `db:"perms"`
		CorrespondingRoles []byte      `db:"corresponding_roles"`
	}

	positionRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[positionDBRow])

	if err != nil {
		return fmt.Errorf("Error while getting staff positions: %v", err)
	}

	posByID := make(map[string]cachedPosition, len(positionRows))
	posByRoleID := make(map[string]cachedPosition, len(positionRows))
	posByName := make(map[string]cachedPosition, len(positionRows))

	for _, p := range positionRows {
		var links []types.Link

		if err := json.Unmarshal(p.CorrespondingRoles, &links); err != nil {
			return err
		}

		pos := cachedPosition{
			ID:                 impls.UUIDString(p.ID),
			Name:               p.Name,
			RoleID:             p.RoleID,
			Index:              p.Index,
			Perms:              p.Perms,
			CorrespondingRoles: links,
		}

		posByID[pos.ID] = pos
		posByRoleID[pos.RoleID] = pos
		posByName[pos.Name] = pos
	}

	// Step 3: load staff members FOR UPDATE.
	rows, err = tx.Query(ctx, "SELECT user_id, positions, perm_overrides, no_autosync, unaccounted FROM staff_members FOR UPDATE")

	if err != nil {
		return fmt.Errorf("Error while getting staff members: %v", err)
	}

	type staffRow struct {
		UserID        string        `db:"user_id"`
		Positions     []pgtype.UUID `db:"positions"`
		PermOverrides []string      `db:"perm_overrides"`
		NoAutosync    bool          `db:"no_autosync"`
		Unaccounted   bool          `db:"unaccounted"`
	}

	staff, err := pgx.CollectRows(rows, pgx.RowToStructByName[staffRow])

	if err != nil {
		return fmt.Errorf("Error while getting staff members: %v", err)
	}

	overridePerms := make(map[string][]perms.Permission, len(staff))
	noAutosync := make(map[string]struct{})
	knownUnaccounted := make(map[string]struct{})
	unaccountedUserIDs := make(map[string]struct{})
	memberPosCache := make(map[string][]string)

	for _, member := range staff {
		overridePerms[member.UserID] = perms.PFSS(member.PermOverrides)

		if member.NoAutosync {
			noAutosync[member.UserID] = struct{}{}
			continue
		}

		if member.Unaccounted {
			knownUnaccounted[member.UserID] = struct{}{}
		}

		unaccountedUserIDs[member.UserID] = struct{}{}
		memberPosCache[member.UserID] = impls.UUIDStrings(member.Positions)
	}

	// Step 4: reconcile each Discord staff member.
	for _, user := range staffResync {
		userID := user.UserID.String()

		if _, skip := noAutosync[userID]; skip {
			continue
		}

		dbPositions, isOnDB := memberPosCache[userID]

		currentPositions := make(map[string]struct{})

		for _, posID := range dbPositions {
			// Garbage collection: drop position ids that no longer exist.
			if _, ok := posByID[posID]; !ok {
				_, err := tx.Exec(ctx,
					"UPDATE staff_members SET positions = array_remove(positions, $1) WHERE user_id = $2",
					posID, userID)

				if err != nil {
					return fmt.Errorf("Error while removing staff member position: %v", err)
				}

				continue
			}

			currentPositions[posID] = struct{}{}
		}

		rolePositions := make(map[string]struct{})

		// Special case: config owners always hold the position named "owner".
		if isOwner(user.UserID) {
			if ownerPos, ok := posByName["owner"]; ok {
				rolePositions[ownerPos.ID] = struct{}{}
			}
		}

		for _, role := range user.Roles {
			pos, ok := posByRoleID[role]

			if !ok {
				continue
			}

			// "owner" is owners-list-driven only and never derived from roles.
			if pos.Name == "owner" {
				continue
			}

			rolePositions[pos.ID] = struct{}{}
		}

		if !differs(rolePositions, currentPositions) {
			delete(unaccountedUserIDs, userID)
			continue
		}

		newPositionIDs := sortedKeys(rolePositions)

		if isOnDB {
			_, err = tx.Exec(ctx,
				"UPDATE staff_members SET positions = $1, unaccounted = false WHERE user_id = $2",
				newPositionIDs, userID)

			if err != nil {
				return fmt.Errorf("Error while updating staff member positions: %v", err)
			}
		} else {
			_, err = tx.Exec(ctx,
				"INSERT INTO staff_members (user_id, positions) VALUES ($1, $2)",
				userID, newPositionIDs)

			if err != nil {
				return fmt.Errorf("Error while inserting staff member positions: %v", err)
			}
		}

		oldSP := buildPermissions(posByID, sortedKeys(currentPositions), overridePerms[userID])
		newSP := buildPermissions(posByID, newPositionIDs, overridePerms[userID])

		var exists bool

		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", userID).Scan(&exists)

		if err != nil {
			return fmt.Errorf("Error while checking if user exists: %v", err)
		}

		if !exists {
			_, err := tx.Exec(ctx, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", userID, impls.GenRandom(512))

			if err != nil {
				return fmt.Errorf("Error while inserting user: %v", err)
			}
		}

		err = impls.SendChannel(state.Config.Channels.StaffLogs, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "Staff Permissions Resync",
				Description: fmt.Sprintf("Updated staff permissions for <@%s>", userID),
				Fields: []discord.EmbedField{
					{Name: "Old Positions", Value: renderPositions(posByID, sortedKeys(currentPositions)), Inline: impls.InlineFalse()},
					{Name: "New Positions", Value: renderPositions(posByID, newPositionIDs), Inline: impls.InlineFalse()},
					{Name: "Old Permissions", Value: renderPerms(oldSP.Resolve()), Inline: impls.InlineFalse()},
					{Name: "New Permissions", Value: renderPerms(newSP.Resolve()), Inline: impls.InlineFalse()},
				},
			}},
		})

		if err != nil {
			return fmt.Errorf("Error while sending staff logs message: %v", err)
		}

		if err := modifyCorrespondingRoles(posByID, user.UserID, sortedKeys(currentPositions), newPositionIDs); err != nil {
			return err
		}

		delete(unaccountedUserIDs, userID)
	}

	// Step 5: handle everyone left in the working set.
	for _, userID := range sortedKeys(unaccountedUserIDs) {
		if _, skip := noAutosync[userID]; skip {
			continue
		}

		if _, skip := knownUnaccounted[userID]; skip {
			continue
		}

		// A member with permission overrides is kept (blanked and flagged); one
		// without is deleted outright.
		remove := len(overridePerms[userID]) == 0

		if remove {
			if _, err := tx.Exec(ctx, "DELETE FROM staff_members WHERE user_id = $1", userID); err != nil {
				return fmt.Errorf("Error while removing unaccounted staff member: %v", err)
			}
		} else {
			_, err := tx.Exec(ctx, "UPDATE staff_members SET positions = '{}', unaccounted = true WHERE user_id = $1", userID)

			if err != nil {
				return fmt.Errorf("Error while updating unaccounted staff member: %v", err)
			}
		}

		// The Rust code unwraps this lookup, which panics for a user that was in
		// the DB but filtered out of the cache. A missing entry is treated as "no
		// positions" here and logged.
		oldPositions, ok := memberPosCache[userID]

		if !ok {
			state.Logger.Warn("Unaccounted staff member missing from position cache", zap.String("userID", userID))
		}

		oldSP := buildPermissions(posByID, oldPositions, overridePerms[userID])

		description := fmt.Sprintf(
			"Updated unaccounted staff member <@%s> as they are no longer in the staff server but have permission overrides.", userID)

		if remove {
			description = fmt.Sprintf(
				"Removed unaccounted staff member <@%s> as they are no longer in the staff server.", userID)
		}

		err = impls.SendChannel(state.Config.Channels.StaffLogs, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "Staff Permissions Resync",
				Description: description,
				Fields: []discord.EmbedField{
					{Name: "Old Positions", Value: renderPositions(posByID, oldPositions), Inline: impls.InlineFalse()},
					{Name: "Old Permissions", Value: renderPerms(oldSP.Resolve()), Inline: impls.InlineFalse()},
				},
			}},
		})

		if err != nil {
			return fmt.Errorf("Error while sending staff logs message: %v", err)
		}

		userSnow, err := snowflake.Parse(userID)

		if err != nil {
			state.Logger.Warn("Unaccounted staff member has an unparseable id", zap.String("userID", userID))
			continue
		}

		if err := modifyCorrespondingRoles(posByID, userSnow, oldPositions, nil); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Error while committing transaction: %v", err)
	}

	return nil
}

func isOwner(userID snowflake.ID) bool {
	for _, owner := range state.Config.Arcadia.Owners {
		if owner == userID {
			return true
		}
	}

	return false
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

func buildPermissions(posByID map[string]cachedPosition, positionIDs []string, overrides []perms.Permission) perms.StaffPermissions {
	sp := perms.StaffPermissions{
		UserPositions: make([]perms.PartialStaffPosition, 0, len(positionIDs)),
		PermOverrides: overrides,
	}

	for _, id := range positionIDs {
		pos, ok := posByID[id]

		if !ok {
			continue
		}

		sp.UserPositions = append(sp.UserPositions, perms.PartialStaffPosition{
			ID:    pos.ID,
			Index: pos.Index,
			Perms: perms.PFSS(pos.Perms),
		})
	}

	return sp
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

func renderPerms(resolved []perms.Permission) string {
	lines := make([]string, 0, len(resolved))

	for _, perm := range resolved {
		lines = append(lines, fmt.Sprintf("- ``%s``", perm.String()))
	}

	if len(lines) == 0 {
		return "None"
	}

	return strings.Join(lines, "\n")
}

// modifyCorrespondingRoles mirrors position changes into the other guilds.
//
// INCONSISTENCY (reproduced): only "main" and "staff" are handled here, while
// the panel's position validator (§5.17) also accepts "testing". A position with
// a "testing" corresponding role validates but is silently ignored by this sync.
// See CONFORMANCE.md.
func modifyCorrespondingRoles(posByID map[string]cachedPosition, user snowflake.ID, removeIDs, addIDs []string) error {
	remove := collectCorrespondingRoles(posByID, removeIDs)
	add := collectCorrespondingRoles(posByID, addIDs)

	for guildID, roles := range remove {
		if !guildMemberPresent(guildID, user) {
			continue
		}

		for _, roleID := range roles {
			if err := impls.RemoveRole(guildID, user, roleID, "Removing corresponding role"); err != nil {
				return err
			}
		}
	}

	for guildID, roles := range add {
		if !guildMemberPresent(guildID, user) {
			continue
		}

		for _, roleID := range roles {
			if err := impls.AddRole(guildID, user, roleID, "Adding corresponding role"); err != nil {
				return err
			}
		}
	}

	return nil
}

func collectCorrespondingRoles(posByID map[string]cachedPosition, positionIDs []string) map[snowflake.ID][]snowflake.ID {
	out := make(map[snowflake.ID][]snowflake.ID)

	for _, id := range positionIDs {
		pos, ok := posByID[id]

		if !ok {
			continue
		}

		for _, link := range pos.CorrespondingRoles {
			var guildID snowflake.ID

			switch link.Name {
			case "main":
				guildID = state.Config.Servers.Main
			case "staff":
				guildID = state.Config.Servers.Staff
			default:
				state.Logger.Warn("Unknown corresponding server", zap.String("name", link.Name))
				continue
			}

			roleID, err := snowflake.Parse(link.Value)

			if err != nil {
				state.Logger.Warn("Unparseable corresponding role id", zap.String("value", link.Value))
				continue
			}

			out[guildID] = append(out[guildID], roleID)
		}
	}

	return out
}

func guildMemberPresent(guildID, user snowflake.ID) bool {
	if _, ok := state.Discord.Caches().Guild(guildID); !ok {
		state.Logger.Warn("Failed to get guild", zap.String("guildID", guildID.String()))
		return false
	}

	if !impls.MemberOnGuild(guildID, user) {
		state.Logger.Warn("User not found in server", zap.String("userID", user.String()))
		return false
	}

	return true
}
