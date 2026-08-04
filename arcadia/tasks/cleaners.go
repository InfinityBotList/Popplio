package tasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// genericEntities maps each entity kind onto its table, id column and the
// target_type value that references it.
var genericEntities = []struct {
	Table      string
	IDColumn   string
	TargetType string
}{
	{"bots", "bot_id", "bot"},
	{"servers", "server_id", "server"},
	{"teams", "id", "team"},
	{"packs", "url", "pack"},
}

// GenericCleaner deletes rows in any table with a target_id column whose target
// no longer exists.
//
// HARDENING (§14b): upstream interpolates information_schema output straight
// into SQL. Here the discovered table name is checked against the set of tables
// that actually carry a target_id column and then quoted with
// pgx.Identifier.Sanitize before use, so a hostile or unexpected table name
// cannot become SQL. The set of tables acted on is unchanged.
func GenericCleaner(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, "select table_name from information_schema.columns where column_name = 'target_id' and table_schema = 'public'")

	if err != nil {
		return err
	}

	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return err
	}

	for _, table := range tables {
		if !validIdentifier(table) {
			state.Logger.Warn("Skipping table with an unexpected name", zap.String("table", table))
			continue
		}

		state.Logger.Info("Validating generic table", zap.String("table", table))

		if err := cleanTable(ctx, table); err != nil {
			return err
		}
	}

	return nil
}

// validIdentifier accepts only the lowercase snake_case identifiers Postgres
// tables in this schema use.
func validIdentifier(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}

	for i, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
		case c == '_':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}

	return true
}

func cleanTable(ctx context.Context, table string) error {
	quoted := pgx.Identifier{table}.Sanitize()

	for _, entity := range genericEntities {
		entityTable := pgx.Identifier{entity.Table}.Sanitize()
		idColumn := pgx.Identifier{entity.IDColumn}.Sanitize()

		query := fmt.Sprintf(
			"select target_id, target_type from %s where target_type = $1 and not exists (select 1 from %s where %s::text = target_id)",
			quoted, entityTable, idColumn,
		)

		rows, err := state.Pool.Query(ctx, query, entity.TargetType)

		if err != nil {
			return fmt.Errorf("Error fetching rows: %s %s", err, query)
		}

		type orphan struct {
			TargetID   string `db:"target_id"`
			TargetType string `db:"target_type"`
		}

		orphans, err := pgx.CollectRows(rows, pgx.RowToStructByName[orphan])

		if err != nil {
			return fmt.Errorf("Error fetching rows: %s %s", err, query)
		}

		for _, o := range orphans {
			state.Logger.Info("Found orphaned generic entity",
				zap.String("table", table), zap.String("targetID", o.TargetID), zap.String("targetType", o.TargetType))

			del := fmt.Sprintf("delete from %s where target_id = $1 and target_type = $2", quoted)

			if _, err := state.Pool.Exec(ctx, del, o.TargetID, o.TargetType); err != nil {
				return fmt.Errorf("Error deleting orphaned generic entity: %s", err)
			}

			state.Logger.Info("Deleted orphaned generic entity",
				zap.String("table", table), zap.String("targetID", o.TargetID), zap.String("targetType", o.TargetType))
		}
	}

	return nil
}

// TeamCleaner deletes empty teams and repairs teams with no global owner or no
// data holder.
func TeamCleaner(ctx context.Context) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "SELECT id FROM teams")

	if err != nil {
		return fmt.Errorf("Error while fetching all teams: %s", err)
	}

	teamIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])

	if err != nil {
		return fmt.Errorf("Error while fetching all teams: %s", err)
	}

	state.Logger.Info("Found teams", zap.Int("count", len(teamIDs)))

	for _, teamID := range teamIDs {
		var memberCount int64

		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM team_members WHERE team_id = $1", teamID).Scan(&memberCount)

		if err != nil {
			return fmt.Errorf("Error while checking if team %s has members: %s", teamID, err)
		}

		if memberCount == 0 {
			if _, err := tx.Exec(ctx, "DELETE FROM teams WHERE id = $1", teamID); err != nil {
				return fmt.Errorf("Error while deleting team %s: %s", teamID, err)
			}

			state.Logger.Info("Deleted team", zap.String("teamID", teamID.String()))
			continue
		}

		globalOwnerRows, err := tx.Query(ctx, "SELECT user_id FROM team_members WHERE team_id = $1 AND flags @> ARRAY[$2]", teamID, string(perms.EntityOwner))

		if err != nil {
			return fmt.Errorf("Error while checking count of team_members with global owner: %s: %s", teamID, err)
		}

		globalOwners, err := pgx.CollectRows(globalOwnerRows, pgx.RowTo[string])

		if err != nil {
			return fmt.Errorf("Error while checking count of team_members with global owner: %s: %s", teamID, err)
		}

		if len(globalOwners) == 0 {
			userID, err := promotionCandidate(ctx, tx, teamID)

			if err != nil {
				return err
			}

			_, err = tx.Exec(ctx,
				"UPDATE team_members SET flags = $1, data_holder = $2 WHERE team_id = $3 AND user_id = $4",
				[]string{string(perms.EntityOwner)}, true, teamID, userID)

			if err != nil {
				return fmt.Errorf("Error while updating flags for team %s: %s", teamID, err)
			}
		}

		var dataHolders int64

		err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND data_holder = true", teamID).Scan(&dataHolders)

		if err != nil {
			return fmt.Errorf("Error while validating data holders of %s: %s", teamID, err)
		}

		if dataHolders == 0 {
			if len(globalOwners) == 0 {
				state.Logger.Warn("Team has no data holders and no global owners", zap.String("teamID", teamID.String()))
				continue
			}

			_, err := tx.Exec(ctx,
				"UPDATE team_members SET data_holder = true WHERE team_id = $1 AND user_id = $2",
				teamID, globalOwners[0])

			if err != nil {
				return fmt.Errorf("Error while updating data_holder for team %s: %s", teamID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Error while committing transaction: %v", err)
	}

	return nil
}

// promotionCandidate picks the member to promote to global owner: the existing
// data holder if there is one, otherwise the first member.
func promotionCandidate(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (string, error) {
	var userID string

	err := tx.QueryRow(ctx, "SELECT user_id FROM team_members WHERE team_id = $1 AND data_holder = true", teamID).Scan(&userID)

	if err == nil {
		return userID, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("Error while fetching data_holder for team %s: %s", teamID, err)
	}

	err = tx.QueryRow(ctx, "SELECT user_id FROM team_members WHERE team_id = $1 LIMIT 1", teamID).Scan(&userID)

	if err != nil {
		return "", fmt.Errorf("Error while fetching first team member for team %s: %s", teamID, err)
	}

	return userID, nil
}
