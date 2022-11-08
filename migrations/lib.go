package migrations

import (
	"context"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
)

func tableExists(ctx context.Context, pool *pgxpool.Pool, name string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", name).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func colExists(ctx context.Context, pool *pgxpool.Pool, table, col string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)", table, col).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func isNone(v pgtype.Text) bool {
	return v.Status == pgtype.Null || v.String == "" || strings.ToLower(v.String) == "none" || strings.ToLower(v.String) == "null"
}

type migrator struct {
	name string
	fn   func(context.Context, *pgxpool.Pool)
}
