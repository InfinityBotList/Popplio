package assets

import (
	"fmt"
	"popplio/state"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigFastest

func DataTask(taskId string, id string, ip string, del bool) {
	l, _ := newTaskLogger(taskId)

	l.Info("Starting DR/DDR task", zap.String("id", id), zap.Bool("del", del))

	tableRefs, err := getAllTableRefs()

	if err != nil {
		l.Error("Failed to get table refs", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	collectedData := map[string]any{}

	for _, tableRef := range tableRefs {
		if _, ok := collectedData[tableRef.TableName]; ok {
			l.Error("Duplicate table. Ignoring/skipping", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id))
			continue
		}

		l.Info("Handling table", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id))

		fkTableOps, ok := tablesOps[tableRef.ForeignTable]

		if !ok {
			l.Warn("Failed to get table ops for foreign table", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id))
		}

		defaultOp := fkTableOps["default"]
		tableOp := fkTableOps[tableRef.TableName]

		if tableOp.Fetch == nil {
			if defaultOp.Fetch == nil {
				l.Warn("Failed to get fetch op for table", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id))

				tableOp.Fetch = func(l *zap.Logger, tableName, columnName, id string) (any, error) {
					return map[string]string{
						"message": fmt.Sprintf("Failed to get fetch op for table %s with column name %s and id %s", tableName, columnName, id),
					}, nil
				}
			} else {
				tableOp.Fetch = defaultOp.Fetch
			}
		}

		rows, err := tableOp.Fetch(l, tableRef.TableName, tableRef.ColumnName, id)

		if err != nil {
			l.Error("Failed to fetch table", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id), zap.Error(err))
		} else {
			collectedData[tableRef.TableName] = rows
		}

		if del {
			if tableOp.Delete == nil {
				if defaultOp.Delete == nil {
					l.Warn("Failed to get delete op for table", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id))

					tableOp.Delete = func(l *zap.Logger, tableName, columnName, id string) error {
						return nil
					}
				} else {
					tableOp.Delete = defaultOp.Delete
				}
			}

			err = tableOp.Delete(l, tableRef.TableName, tableRef.ColumnName, id)

			if err != nil {
				l.Error("Failed to delete table", zap.String("table", tableRef.TableName), zap.String("foreignTable", tableRef.ForeignTable), zap.String("column", tableRef.ColumnName), zap.String("id", id), zap.Error(err))
			}
		}
	}

	// Delete from psql user_cache if `del` is true
	if del {
		for _, dovewingPlatform := range []*dovewing.DiscordState{state.DovewingPlatformDiscord} {
			l.Info("Deleting from user_cache [dovewing]", zap.String("id", id), zap.String("platform", dovewingPlatform.PlatformName()))
			res, err := dovewing.ClearUser(state.Context, id, dovewingPlatform, dovewing.ClearUserReq{})

			if err != nil {
				l.Error("Error clearing user [dovewing]", zap.Error(err), zap.String("id", id), zap.String("platform", dovewingPlatform.PlatformName()))
			}

			l.Info("Cleared user [dovewing]", zap.String("id", id), zap.String("platform", dovewingPlatform.PlatformName()), zap.Any("res", res))
		}
	}

	bytes, err := json.Marshal(collectedData)

	if err != nil {
		l.Error("Failed to encode final collected data", zap.Error(err), zap.String("id", id), zap.Bool("del", del))
		return
	}

	state.Redis.Set(state.Context, "task:output:"+taskId, string(bytes), 15*time.Minute)
}
