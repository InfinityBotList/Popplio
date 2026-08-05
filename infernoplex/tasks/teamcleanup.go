package tasks

import (
	"context"

	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

func TeamCleanup(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, `
		SELECT tm.team_id, tm.user_id, s.server_id
		FROM team_members tm
		JOIN servers s ON s.team_owner = tm.team_id
		WHERE tm.service = 'infernoplex'
	`)

	if err != nil {
		return err
	}

	type memberRow struct {
		teamID   uuid.UUID
		userID   string
		serverID string
	}

	var memberRows []memberRow

	for rows.Next() {
		var r memberRow

		if err := rows.Scan(&r.teamID, &r.userID, &r.serverID); err != nil {
			rows.Close()
			return err
		}

		memberRows = append(memberRows, r)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range memberRows {
		guildID, err := snowflake.Parse(r.serverID)

		if err != nil {
			continue
		}

		userID, err := snowflake.Parse(r.userID)

		if err != nil {
			continue
		}

		member, err := dclient.Get().Rest().GetMember(guildID, userID)

		if err != nil {

			if err := removeTeamMember(ctx, r.teamID, r.userID); err != nil {
				return err
			}

			continue
		}

		guild, err := dclient.Get().Rest().GetGuild(guildID, false)

		if err != nil {
			continue
		}

		isAdmin := member.User.ID == guild.OwnerID || hasAdministratorRole(member.RoleIDs, guild.Roles)

		if !isAdmin {
			if err := removeTeamMember(ctx, r.teamID, r.userID); err != nil {
				return err
			}
		}
	}

	return nil
}

func hasAdministratorRole(roleIDs []snowflake.ID, roles []discord.Role) bool {
	byID := make(map[snowflake.ID]discord.Role, len(roles))

	for _, role := range roles {
		byID[role.ID] = role
	}

	for _, id := range roleIDs {
		if role, ok := byID[id]; ok && role.Permissions.Has(discord.PermissionAdministrator) {
			return true
		}
	}

	return false
}

func removeTeamMember(ctx context.Context, teamID uuid.UUID, userID string) error {
	_, err := state.Pool.Exec(ctx,
		"DELETE FROM team_members WHERE team_id = $1 AND user_id = $2 AND service = 'infernoplex'",
		teamID, userID,
	)

	return err
}
