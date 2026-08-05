package tasks

import (
	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// Mirroring a position's "corresponding roles" into the other guilds.
//
// A position can carry Discord roles in the main and staff servers that follow
// it: gaining the position adds them, losing it takes them away. This is the
// only part of the resync that writes to Discord rather than to the database.

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
	if _, ok := dclient.Get().Caches().Guild(guildID); !ok {
		state.Logger.Warn("Failed to get guild", zap.String("guildID", guildID.String()))
		return false
	}

	if !impls.MemberOnGuild(guildID, user) {
		state.Logger.Warn("User not found in server", zap.String("userID", user.String()))
		return false
	}

	return true
}
