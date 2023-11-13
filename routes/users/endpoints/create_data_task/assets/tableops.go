package assets

import (
	"fmt"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func defaultFetchOp(l *zap.Logger, tableName, columnName, id string) (any, error) {
	l.Info("Fetching table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	rows, err := state.Pool.Query(state.Context, "SELECT * FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return nil, fmt.Errorf("failed to query table: %w", err)
	}

	return pgx.CollectRows(rows, pgx.RowToMap)
}

func defaultDeleteOp(l *zap.Logger, tableName, columnName, id string) error {
	l.Info("Deleting from table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	_, err := state.Pool.Exec(state.Context, "DELETE FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return fmt.Errorf("failed to delete from table: %w", err)
	}

	return nil
}

type TableOps struct {
	Fetch  func(l *zap.Logger, tableName, columnName, id string) (any, error)
	Delete func(l *zap.Logger, tableName, columnName, id string) error
}

// Map of foreign tablenames to a map of tablenames to table operations
var tablesOps = map[string]map[string]TableOps{
	"users": {
		"default": {
			Fetch:  defaultFetchOp,
			Delete: defaultDeleteOp,
		},
	},
	"default": {
		"default": {
			Fetch:  defaultFetchOp,
			Delete: defaultDeleteOp,
		},
	},
}
