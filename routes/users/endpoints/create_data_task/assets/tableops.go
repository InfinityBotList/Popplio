package assets

import (
	"fmt"
	"popplio/state"
	"popplio/teams"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func defaultFetcher(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) ([]map[string]any, error) {
	l.Info("Fetching table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	rows, err := tx.Query(state.Context, "SELECT * FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return nil, fmt.Errorf("failed to query table: %w", err)
	}

	return pgx.CollectRows(rows, pgx.RowToMap)
}

func defaultDeleter(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) error {
	l.Info("Deleting from table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	_, err := tx.Exec(state.Context, "DELETE FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return fmt.Errorf("failed to delete from table: %w", err)
	}

	return nil
}

// Table Logic represents a list of foreign keys for a table
type TableLogic struct {
	// GetIdsForUser returns a list of IDs for the user that should be fetched/deleted
	GetIdsForUser func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error)

	// Fetch returns a list of rows for the given table, column and ID
	Fetch func(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) ([]map[string]any, error)

	// Delete deletes all rows for the given table, column and ID
	Delete func(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) error
}

// Table specific converters
//
// This is run on the table name, not the foreign key
type TableTransformers struct {
	// Convert converts data to the final data for a fetch. This is called on both 'data_request' and 'data_delete'
	//
	// This is useful when having secrets in the database that should not be exposed
	Fetch []func(data []map[string]any) ([]map[string]any, error)
}

var tableLogic = map[string]TableLogic{
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
		Fetch:  defaultFetcher,
		Delete: defaultDeleter,
	},
	"users": {
		GetIdsForUser: func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error) {
			return []string{id}, nil
		},
		Fetch:  defaultFetcher,
		Delete: defaultDeleter,
	},
}

var tableTransformer = map[string]TableTransformers{
	"staffpanel__authchain": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				for i := range data {
					// Redact tokens to avoid leaking them
					data[i]["token"] = "REDACTED"
					data[i]["popplio_token"] = "REDACTED"
				}

				return data, nil
			},
		},
	},
	"staffpanel__paneldata": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				for i := range data {
					// Redact MFA tokens to avoid leaking them
					data[i]["mfa_secret"] = "REDACTED"
					data[i]["mfa_verified"] = "REDACTED"
				}

				return data, nil
			},
		},
	},
	"bots": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				for i := range data {
					// Redact tokens to avoid leaking them but only if theyre not in a team
					if data[i]["team_owner"] == nil {
						data[i]["api_token"] = "REDACTED_AS_IN_A_TEAM"
					}

					// Format itag as a UUID
					data[i]["itag"] = fmt.Sprintf("%x-%x-%x-%x", data[i]["itag"].([]byte)[0:4], data[i]["itag"].([]byte)[4:6], data[i]["itag"].([]byte)[6:8], data[i]["itag"].([]byte)[8:10])
				}

				return data, nil
			},
		},
	},
}
