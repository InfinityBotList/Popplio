package impls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"popplio/arcadia/types"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// EntityManagers is the set of users who own/manage an entity.
type EntityManagers struct {
	users []manager
}

type manager struct {
	mentionable bool
	user        string
}

// All returns every manager's user id.
func (e EntityManagers) All() []string {
	all := make([]string, 0, len(e.users))
	for _, m := range e.users {
		all = append(all, m.user)
	}

	return all
}

// Mentionables returns only the managers flagged mentionable.
func (e EntityManagers) Mentionables() []string {
	out := make([]string, 0, len(e.users))
	for _, m := range e.users {
		if m.mentionable {
			out = append(out, m.user)
		}
	}

	return out
}

// MentionUsers renders the mentionable managers as a ", "-joined mention list.
func (e EntityManagers) MentionUsers() string {
	out := make([]string, 0, len(e.users))
	for _, m := range e.users {
		if m.mentionable {
			out = append(out, "<@"+m.user+">")
		}
	}

	return strings.Join(out, ", ")
}

// GetEntityManagers resolves who manages a given entity. Error strings are
// user-visible in Discord and the panel, so they are reproduced verbatim.
func GetEntityManagers(ctx context.Context, targetType types.TargetType, targetID string) (EntityManagers, error) {
	var teamID uuid.UUID

	switch targetType {
	case types.TargetTypeBot:
		var owner *string

		err := state.Pool.QueryRow(ctx, "SELECT owner FROM bots WHERE bot_id = $1", targetID).Scan(&owner)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while checking for owner of bot %s: %s", targetID, err)
		}

		if owner != nil {
			return EntityManagers{users: []manager{{mentionable: true, user: *owner}}}, nil
		}

		var teamOwner *uuid.UUID

		err = state.Pool.QueryRow(ctx, "SELECT team_owner FROM bots WHERE bot_id = $1", targetID).Scan(&teamOwner)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while checking for team owner of bot %s: %s", targetID, err)
		}

		if teamOwner == nil {
			return EntityManagers{}, fmt.Errorf("Bot %s is not owned by a team or a user. Please contact a dev right now!", targetID)
		}

		teamID = *teamOwner
	case types.TargetTypeServer:
		var teamOwner uuid.UUID

		err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM servers WHERE server_id = $1", targetID).Scan(&teamOwner)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while checking for team owner of server %s: %s", targetID, err)
		}

		teamID = teamOwner
	case types.TargetTypeTeam:
		parsed, err := uuid.Parse(targetID)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while parsing team id %s: %s", targetID, err)
		}

		teamID = parsed
	case types.TargetTypeUser:
		var userID string

		err := state.Pool.QueryRow(ctx, "SELECT user_id FROM users WHERE user_id = $1", targetID).Scan(&userID)

		switch {
		case err == nil:
			return EntityManagers{users: []manager{{mentionable: true, user: userID}}}, nil
		case errors.Is(err, pgx.ErrNoRows):
			return EntityManagers{}, fmt.Errorf("User %s not found.", targetID)
		default:
			return EntityManagers{}, fmt.Errorf("Error while checking for user %s: %s", targetID, err)
		}
	case types.TargetTypePack:
		return EntityManagers{}, errors.New("Packs are not supported yet!")
	default:
		return EntityManagers{}, errors.New("Packs are not supported yet!")
	}

	rows, err := state.Pool.Query(ctx, "SELECT user_id, mentionable FROM team_members WHERE team_id = $1", teamID)

	if err != nil {
		return EntityManagers{}, fmt.Errorf("Error while getting team members of team %s: %s", teamID, err)
	}

	type teamMember struct {
		UserID      string `db:"user_id"`
		Mentionable bool   `db:"mentionable"`
	}

	members, err := pgx.CollectRows(rows, pgx.RowToStructByName[teamMember])

	if err != nil {
		return EntityManagers{}, fmt.Errorf("Error while getting team members of team %s: %s", teamID, err)
	}

	if len(members) == 0 {
		return EntityManagers{}, fmt.Errorf("Entity %s is on a team with no members. Please contact a dev right now!", targetID)
	}

	users := make([]manager, 0, len(members))
	for _, m := range members {
		users = append(users, manager{mentionable: m.Mentionable, user: m.UserID})
	}

	return EntityManagers{users: users}, nil
}

// GetBotManagers resolves managers for many bots in a fixed number of queries.
//
// GetEntityManagers costs two round trips per bot, so the panel's bot queue and
// entity search were issuing 2N queries for an N-bot response. This resolves the
// same data with three queries regardless of N.
//
// The per-bot error messages are reproduced exactly, so a caller cannot tell the
// difference between this and calling GetEntityManagers in a loop.
func GetBotManagers(ctx context.Context, botIDs []string) (map[string]EntityManagers, error) {
	out := make(map[string]EntityManagers, len(botIDs))

	if len(botIDs) == 0 {
		return out, nil
	}

	rows, err := state.Pool.Query(ctx, "SELECT bot_id, owner, team_owner FROM bots WHERE bot_id = ANY($1)", botIDs)

	if err != nil {
		return nil, fmt.Errorf("Error while checking for owner of bots: %s", err)
	}

	type ownerRow struct {
		BotID     string     `db:"bot_id"`
		Owner     *string    `db:"owner"`
		TeamOwner *uuid.UUID `db:"team_owner"`
	}

	owners, err := pgx.CollectRows(rows, pgx.RowToStructByName[ownerRow])

	if err != nil {
		return nil, fmt.Errorf("Error while checking for owner of bots: %s", err)
	}

	byBot := make(map[string]ownerRow, len(owners))
	teamIDs := make([]uuid.UUID, 0, len(owners))

	for _, row := range owners {
		byBot[row.BotID] = row

		if row.Owner == nil && row.TeamOwner != nil {
			teamIDs = append(teamIDs, *row.TeamOwner)
		}
	}

	teams, err := teamMembers(ctx, teamIDs)

	if err != nil {
		return nil, err
	}

	for _, botID := range botIDs {
		row, ok := byBot[botID]

		if !ok {
			return nil, fmt.Errorf("Error while checking for owner of bot %s: %s", botID, pgx.ErrNoRows)
		}

		if row.Owner != nil {
			out[botID] = EntityManagers{users: []manager{{mentionable: true, user: *row.Owner}}}
			continue
		}

		if row.TeamOwner == nil {
			return nil, fmt.Errorf("Bot %s is not owned by a team or a user. Please contact a dev right now!", botID)
		}

		members := teams[*row.TeamOwner]

		if len(members) == 0 {
			return nil, fmt.Errorf("Entity %s is on a team with no members. Please contact a dev right now!", botID)
		}

		out[botID] = EntityManagers{users: members}
	}

	return out, nil
}

// GetServerManagers is the server equivalent of GetBotManagers.
func GetServerManagers(ctx context.Context, serverIDs []string) (map[string]EntityManagers, error) {
	out := make(map[string]EntityManagers, len(serverIDs))

	if len(serverIDs) == 0 {
		return out, nil
	}

	rows, err := state.Pool.Query(ctx, "SELECT server_id, team_owner FROM servers WHERE server_id = ANY($1)", serverIDs)

	if err != nil {
		return nil, fmt.Errorf("Error while checking for team owner of servers: %s", err)
	}

	type ownerRow struct {
		ServerID  string    `db:"server_id"`
		TeamOwner uuid.UUID `db:"team_owner"`
	}

	owners, err := pgx.CollectRows(rows, pgx.RowToStructByName[ownerRow])

	if err != nil {
		return nil, fmt.Errorf("Error while checking for team owner of servers: %s", err)
	}

	byServer := make(map[string]uuid.UUID, len(owners))
	teamIDs := make([]uuid.UUID, 0, len(owners))

	for _, row := range owners {
		byServer[row.ServerID] = row.TeamOwner
		teamIDs = append(teamIDs, row.TeamOwner)
	}

	teams, err := teamMembers(ctx, teamIDs)

	if err != nil {
		return nil, err
	}

	for _, serverID := range serverIDs {
		teamID, ok := byServer[serverID]

		if !ok {
			return nil, fmt.Errorf("Error while checking for team owner of server %s: %s", serverID, pgx.ErrNoRows)
		}

		members := teams[teamID]

		if len(members) == 0 {
			return nil, fmt.Errorf("Entity %s is on a team with no members. Please contact a dev right now!", serverID)
		}

		out[serverID] = EntityManagers{users: members}
	}

	return out, nil
}

// teamMembers loads the members of many teams in one query.
func teamMembers(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID][]manager, error) {
	out := make(map[uuid.UUID][]manager, len(teamIDs))

	if len(teamIDs) == 0 {
		return out, nil
	}

	rows, err := state.Pool.Query(ctx, "SELECT team_id, user_id, mentionable FROM team_members WHERE team_id = ANY($1)", teamIDs)

	if err != nil {
		return nil, fmt.Errorf("Error while getting team members of teams: %s", err)
	}

	type memberRow struct {
		TeamID      uuid.UUID `db:"team_id"`
		UserID      string    `db:"user_id"`
		Mentionable bool      `db:"mentionable"`
	}

	members, err := pgx.CollectRows(rows, pgx.RowToStructByName[memberRow])

	if err != nil {
		return nil, fmt.Errorf("Error while getting team members of teams: %s", err)
	}

	for _, m := range members {
		out[m.TeamID] = append(out[m.TeamID], manager{mentionable: m.Mentionable, user: m.UserID})
	}

	return out, nil
}

// OwnedBy is one entity a user owns.
type OwnedBy struct {
	TargetType  types.TargetType
	TargetID    string
	EntityState string
}

// GetOwnedBy unions the bots and servers owned via the user's teams with the
// packs they own outright. Unknown entity strings are logged and skipped.
func GetOwnedBy(ctx context.Context, userID string) ([]OwnedBy, error) {
	rows, err := state.Pool.Query(ctx, `
        SELECT bot_id as id, type, 'bot' as entity
        FROM bots
        WHERE team_owner IN (SELECT team_id FROM team_members WHERE user_id = $1)

        UNION

        SELECT server_id as id, type, 'server' as entity
        FROM servers
        WHERE team_owner IN (SELECT team_id FROM team_members WHERE user_id = $1)

        UNION

        SELECT url as id, 'pack' as type, 'pack' as entity
        FROM packs
        WHERE owner = $1
        `, userID)

	if err != nil {
		return nil, fmt.Errorf("Error while executing query for user %s: %s", userID, err)
	}

	defer rows.Close()

	var ownedBy []OwnedBy

	for rows.Next() {
		var id, entityState, entity string

		if err := rows.Scan(&id, &entityState, &entity); err != nil {
			return nil, fmt.Errorf("Error while executing query for user %s: %s", userID, err)
		}

		var targetType types.TargetType

		switch entity {
		case "bot":
			targetType = types.TargetTypeBot
		case "server":
			targetType = types.TargetTypeServer
		case "pack":
			targetType = types.TargetTypePack
		default:
			state.Logger.Error("Unknown entity type encountered", zap.String("entity", entity), zap.String("userID", userID))
			continue
		}

		ownedBy = append(ownedBy, OwnedBy{
			TargetType:  targetType,
			TargetID:    id,
			EntityState: entityState,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error while executing query for user %s: %s", userID, err)
	}

	return ownedBy, nil
}
