package assets

import (
	"popplio/state"

	"github.com/infinitybotlist/eureka/dovewing"
	"go.uber.org/zap"
)

type deleteCall struct {
	fOp  TableLogic
	keys [3]string
}

func DataTask(taskId, taskName, id, ip string) {
	del := taskName == "data_delete"

	var done bool

	l, _ := newTaskLogger(taskId)

	// Fail failed tasks
	defer func() {
		err := recover()

		if err != nil {
			l.Error("Panic", zap.Any("err", err), zap.String("id", id), zap.Bool("del", del))

			_, err := state.Pool.Exec(state.Context, "UPDATE tasks SET state = $1 WHERE task_id = $2", "failed", taskId)

			if err != nil {
				l.Error("Failed to update task", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
			}
		}

		if !done {
			l.Error("Failed to complete task", zap.String("id", id), zap.Bool("del", del))

			_, err := state.Pool.Exec(state.Context, "UPDATE tasks SET state = $1 WHERE task_id = $2", "failed", taskId)

			if err != nil {
				l.Error("Failed to update task", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
			}
		}
	}()

	tx, err := state.Pool.Begin(state.Context)

	if err != nil {
		l.Error("Failed to begin transaction", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	defer tx.Rollback(state.Context)

	_, err = tx.Exec(state.Context, "DELETE FROM tasks WHERE task_name = $1 AND task_id != $2 AND for_user = $3", taskName, taskId, id)

	if err != nil {
		l.Error("Failed to delete old data tasks", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	l.Info("Started DR/DDR task", zap.String("id", id), zap.Bool("del", del))

	tableRefs, err := getAllTableRefs(tx)

	if err != nil {
		l.Error("Failed to get table refs", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	// Begin fetching data recursively
	collectedData := map[string][]any{}
	cachedEntityIds := map[string][]string{}

	deleteCalls := []deleteCall{}
	for _, tableRef := range tableRefs {
		fOp, ok := tableLogic[tableRef.ForeignTableName]

		if !ok {
			l.Warn("Cannot fetch table due to no support for its foreign ref", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTableName), zap.String("column", tableRef.ColumnName), zap.String("id", id))
			continue
		}

		var entityIds []string

		if ids, ok := cachedEntityIds[tableRef.ForeignTableName]; ok {
			entityIds = ids
		} else {
			entityIds, err := fOp.GetIdsForUser(tx, l, id)

			if err != nil {
				l.Error("Failed to get ids", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTableName), zap.String("column", tableRef.ColumnName), zap.String("id", id), zap.Error(err))
				continue
			}

			cachedEntityIds[tableRef.ForeignTableName] = entityIds
		}

		var handleKeys = [][3]string{
			{"main", tableRef.TableName, tableRef.ColumnName},
		}
		if _, ok := collectedData[tableRef.ForeignTableName]; !ok {
			handleKeys = append(handleKeys, [3]string{"foreign", tableRef.ForeignTableName, tableRef.ForeignColumnName})
		}

		// Handle the entities now
		for _, entityId := range entityIds {
			for _, handleKey := range handleKeys {
				l.Info("Fetching column for table", zap.String("type", handleKey[0]), zap.String("table", handleKey[1]), zap.String("foreignTable", tableRef.ForeignTableName), zap.String("column", handleKey[2]), zap.String("id", id), zap.String("entityId", entityId))
				rows, err := fOp.Fetch(tx, l, handleKey[1], handleKey[2], entityId)

				if err != nil {
					l.Error("Failed to fetch table", zap.Error(err), zap.String("type", handleKey[0]), zap.String("table", handleKey[1]), zap.String("foreignTable", tableRef.ForeignTableName), zap.String("column", handleKey[2]), zap.String("id", id), zap.String("entityId", entityId))
					continue
				}

				// Run transformers
				for i, transformer := range tableTransformer[handleKey[1]].Fetch {
					rows, err = transformer(rows)

					if err != nil {
						l.Error("Failed to transform table", zap.Error(err), zap.String("transform", handleKey[1]), zap.Int("index", i), zap.String("type", handleKey[0]), zap.String("table", handleKey[1]), zap.String("foreignTable", tableRef.ForeignTableName), zap.String("column", handleKey[2]), zap.String("id", id), zap.String("entityId", entityId))
						continue
					}
				}

				for i, transformer := range tableTransformer["*"].Fetch {
					rows, err = transformer(rows)

					if err != nil {
						l.Error("Failed to apply default transforms", zap.Error(err), zap.String("transform", "*"), zap.Int("index", i), zap.String("type", handleKey[0]), zap.String("table", handleKey[1]), zap.String("foreignTable", tableRef.ForeignTableName), zap.String("column", handleKey[2]), zap.String("id", id), zap.String("entityId", entityId))
						continue
					}
				}

				for _, row := range rows {
					if _, ok := collectedData[handleKey[1]]; !ok {
						collectedData[handleKey[1]] = []any{}
					}

					collectedData[handleKey[1]] = append(collectedData[handleKey[1]], row)
				}

				if del {
					deleteCalls = append(deleteCalls, deleteCall{
						fOp: fOp,
						keys: [3]string{
							handleKey[1],
							handleKey[2],
							entityId,
						},
					})
				}
			}
		}
	}

	finalOutput := map[string]any{
		"data": collectedData,
		"meta": map[string]any{
			"request_ip": ip,
		},
	}

	// Delete from psql user_cache if `del` is true
	if del {
		for _, deleteCall := range deleteCalls {
			if deleteCall.keys[0] == "users" {
				continue
			}
			l.Info("Deleting from table", zap.String("table", deleteCall.keys[0]), zap.String("column", deleteCall.keys[1]), zap.String("id", id), zap.String("entityId", deleteCall.keys[2]))
			err = deleteCall.fOp.Delete(tx, l, deleteCall.keys[0], deleteCall.keys[1], deleteCall.keys[2])

			if err != nil {
				l.Info("Failed to delete from table", zap.Error(err), zap.String("table", deleteCall.keys[0]), zap.String("column", deleteCall.keys[1]), zap.String("id", id), zap.String("entityId", deleteCall.keys[2]))
			}
		}

		_, err = tx.Exec(state.Context, "DELETE FROM users WHERE user_id = $1", id)

		if err != nil {
			l.Error("Failed to delete user", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
			return
		}

		for _, dovewingPlatform := range []dovewing.Platform{state.DovewingPlatformDiscord} {
			l.Info("Deleting from user_cache [dovewing]", zap.String("id", id), zap.String("platform", dovewingPlatform.PlatformName()))
			res, err := dovewing.ClearUser(state.Context, id, dovewingPlatform, dovewing.ClearUserReq{})

			if err != nil {
				l.Error("Error clearing user [dovewing]", zap.Error(err), zap.String("id", id), zap.String("platform", dovewingPlatform.PlatformName()))
			}

			l.Info("Cleared user [dovewing]", zap.String("id", id), zap.String("platform", dovewingPlatform.PlatformName()), zap.Any("res", res))
		}
	}

	_, err = tx.Exec(state.Context, "UPDATE tasks SET output = $1, state = $2 WHERE task_id = $3", finalOutput, "completed", taskId)

	if err != nil {
		l.Error("Failed to update task", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	err = tx.Commit(state.Context)

	if err != nil {
		l.Error("Failed to commit transaction", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	done = true
}
