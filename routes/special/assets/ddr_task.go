package assets

import (
	"popplio/state"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const ddrStr = `
SELECT 
  tbl.relname AS table_name,
  col.attname AS column_name,
  referenced_tbl.relname AS foreign_table_name,
  referenced_field.attname AS foreign_column_name
FROM pg_constraint c
    INNER JOIN pg_namespace AS sh ON sh.oid = c.connamespace
    INNER JOIN (SELECT oid, unnest(conkey) as conkey FROM pg_constraint) con ON c.oid = con.oid
    INNER JOIN pg_class tbl ON tbl.oid = c.conrelid
    INNER JOIN pg_attribute col ON (col.attrelid = tbl.oid AND col.attnum = con.conkey)
    INNER JOIN pg_class referenced_tbl ON c.confrelid = referenced_tbl.oid
    INNER JOIN pg_namespace AS referenced_sh ON referenced_sh.oid = referenced_tbl.relnamespace
    INNER JOIN (SELECT oid, unnest(confkey) as confkey FROM pg_constraint) conf ON c.oid = conf.oid
    INNER JOIN pg_attribute referenced_field ON (referenced_field.attrelid = c.confrelid AND referenced_field.attnum = conf.confkey)
WHERE c.contype = 'f'`

func DataTask(taskId string, id string, ip string, del bool) {
	ctx := state.Context

	state.Redis.SetArgs(ctx, taskId, "Fetching basic user data", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	var keys []*struct {
		ForeignTable      string `db:"foreign_table_name"`
		TableName         string `db:"table_name"`
		ColumnName        string `db:"column_name"`
		ForeignColumnName string `db:"foreign_column_name"`
	}

	data, err := state.Pool.Query(ctx, ddrStr)

	if err != nil {
		state.Logger.Error(err)

		state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	if err := pgxscan.ScanAll(&keys, data); err != nil {
		state.Logger.Error(err)

		state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	finalDump := make(map[string]any)

	for _, key := range keys {
		if key.ForeignTable == "users" {
			sqlStmt := "SELECT * FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

			data, err := state.Pool.Query(ctx, sqlStmt, id)

			if err != nil {
				state.Logger.Error(err)
			}

			var rows []map[string]any

			if err := pgxscan.ScanAll(&rows, data); err != nil {
				state.Logger.Error(err)

				state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
					KeepTTL: true,
				})

				return
			}

			if del {
				sqlStmt = "DELETE FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

				_, err := state.Pool.Exec(ctx, sqlStmt, id)

				if err != nil {
					state.Logger.Error(err)

					state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})

					return
				}
			}

			finalDump[key.TableName] = rows
		}
	}

	// Delete from psql user_cache if `del` is true
	if del {
		_, err := state.Pool.Exec(ctx, "DELETE FROM internal_user_cache WHERE id = $1", id)

		if err != nil {
			// Just log it, don't return as it's not critical
			state.Logger.Error(err)

			finalDump["internal_user_cache"] = map[string]any{
				"message": "Failed to delete from internal_user_cache",
				"error":   true,
				"ctx":     err.Error(),
			}
		} else {
			finalDump["internal_user_cache"] = map[string]any{
				"message": "Successfully deleted from internal_user_cache",
				"error":   false,
				"ctx":     nil,
			}
		}
	}

	bytes, err := json.Marshal(finalDump)

	if err != nil {
		state.Logger.Error("Failed to encode data")
		state.Redis.SetArgs(ctx, taskId, "Failed to encode data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	state.Redis.SetArgs(ctx, taskId, string(bytes), redis.SetArgs{
		KeepTTL: false,
	})
}
