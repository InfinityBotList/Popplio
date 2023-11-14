package assets

import (
	"fmt"
	"popplio/state"
	"popplio/teams"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func defaultFetchOp(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) ([]map[string]any, error) {
	l.Info("Fetching table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	rows, err := tx.Query(state.Context, "SELECT * FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return nil, fmt.Errorf("failed to query table: %w", err)
	}

	return pgx.CollectRows(rows, pgx.RowToMap)
}

func defaultDeleteOp(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) error {
	l.Info("Deleting from table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	_, err := tx.Exec(state.Context, "DELETE FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return fmt.Errorf("failed to delete from table: %w", err)
	}

	return nil
}

type TableOps struct {
	// GetIdsForUser returns a list of IDs for the user that should be fetched/deleted
	GetIdsForUser func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error)
	Fetch         func(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) ([]map[string]any, error)
	Delete        func(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) error
}

var tableOps = map[string]TableOps{
	"teams": {
		GetIdsForUser: func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error) {
			// Select all team IDs
			var teamIds []string

			rows, err := tx.Query(state.Context, "SELECT team_id, flags FROM team_members WHERE user_id = $1", id)

			if err != nil {
				return nil, fmt.Errorf("failed to query table: %w", err)
			}

			for rows.Next() {
				var teamId string
				var flags []string

				err = rows.Scan(&teamId, &flags)

				if err != nil {
					return nil, fmt.Errorf("failed to scan row: %w", err)
				}

				if teams.NewPermMan(flags).HasRaw("global." + teams.PermissionOwner) {
					teamIds = append(teamIds, teamId)
				} else {
					l.Warn("User does not have permission to dump team", zap.String("team_id", teamId), zap.String("user_id", id))
				}
			}

			return teamIds, nil
		},
		Fetch:  defaultFetchOp,
		Delete: defaultDeleteOp,
	},
	"users": {
		GetIdsForUser: func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error) {
			return []string{id}, nil
		},
		Fetch:  defaultFetchOp,
		Delete: defaultDeleteOp,
	},
}
