package migrations

import (
	"kitehelper/common"
	"kitehelper/migrate"
)

var ctx = migrate.Ctx

func tableExists(pool *common.SandboxPool, name string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", name).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func colExists(pool *common.SandboxPool, table, col string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)", table, col).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}
