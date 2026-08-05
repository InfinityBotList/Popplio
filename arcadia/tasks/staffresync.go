package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// The staff resync itself, in the five steps it has always run in.
//
// The steps are deliberately left inline rather than broken into a function
// each: they share a transaction, a working set that each one narrows, and an
// ordering that is load-bearing — step 5 acts on exactly what step 4 did not
// account for. Splitting them would mean passing that state around in a struct
// and would make the order look optional when it is not. What could be lifted
// out without touching that — the reporting, and the Discord role mirroring —
// is in staffresync_report.go and staffresync_roles.go.

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
	if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Staff); !ok {
		return fmt.Errorf("Failed to get staff guild for staff perms resync")
	}

	type guildMember struct {
		UserID snowflake.ID
		Roles  []string
		IsBot  bool
	}

	var staffResync []guildMember

	dclient.Get().Caches().MembersForEach(state.Config.Servers.Staff, func(member discord.Member) {
		roles := make([]string, 0, len(member.RoleIDs))

		for _, roleID := range member.RoleIDs {
			roles = append(roles, roleID.String())
		}

		// The gateway is the authority on what an account is, so bots are
		// recognised here rather than looked up again later.
		staffResync = append(staffResync, guildMember{UserID: member.User.ID, Roles: roles, IsBot: member.User.Bot})
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

	overridePerms := make(map[string][]perms.Perm, len(staff))
	noAutosync := make(map[string]struct{})
	knownUnaccounted := make(map[string]struct{})
	unaccountedUserIDs := make(map[string]struct{})
	memberPosCache := make(map[string][]string)

	for _, member := range staff {
		overridePerms[member.UserID] = perms.ParseStrings(member.PermOverrides)

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

	// bots are the bot accounts sitting in the staff server. They are tracked so
	// that step 5 can say why it is removing one, rather than reporting them as
	// having left a server they are still in.
	bots := make(map[string]struct{})

	// Step 4: reconcile each Discord staff member.
	for _, user := range staffResync {
		userID := user.UserID.String()

		// A bot never becomes a staff member, whatever roles it has been given
		// in the staff server. Leaving it out of the reconcile is also what
		// takes an existing bot's staff row away: step 5 handles everyone the
		// reconcile did not account for.
		if user.IsBot {
			bots[userID] = struct{}{}
			continue
		}

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
		oldPositionIDs := sortedKeys(currentPositions)

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

		oldSP := buildPermissions(posByID, oldPositionIDs, overridePerms[userID])
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

		announceResync(userID, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "Staff Permissions Resync",
				Description: fmt.Sprintf("Updated staff permissions for <@%s>", userID),
				Fields: []discord.EmbedField{
					{Name: "Old Positions", Value: renderPositions(posByID, oldPositionIDs), Inline: impls.InlineFalse()},
					{Name: "New Positions", Value: renderPositions(posByID, newPositionIDs), Inline: impls.InlineFalse()},
					{Name: "Old Permissions", Value: renderPerms(oldSP.Resolve()), Inline: impls.InlineFalse()},
					{Name: "New Permissions", Value: renderPerms(newSP.Resolve()), Inline: impls.InlineFalse()},
				},
			}},
		})

		if err := modifyCorrespondingRoles(posByID, user.UserID, oldPositionIDs, newPositionIDs); err != nil {
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

		// A bot is still in the staff server; it is being removed for being a
		// bot, and saying it left would be wrong as well as confusing.
		_, isBot := bots[userID]

		verb, reason := "Removed", "they are no longer in the staff server"

		if isBot {
			reason = "bot accounts cannot hold staff permissions"
		}

		if !remove {
			verb, reason = "Updated", reason+", but they have permission overrides"
		}

		description := fmt.Sprintf("%s unaccounted staff member <@%s> as %s.", verb, userID, reason)

		announceResync(userID, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "Staff Permissions Resync",
				Description: description,
				Fields: []discord.EmbedField{
					{Name: "Old Positions", Value: renderPositions(posByID, oldPositions), Inline: impls.InlineFalse()},
					{Name: "Old Permissions", Value: renderPerms(oldSP.Resolve()), Inline: impls.InlineFalse()},
				},
			}},
		})

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
