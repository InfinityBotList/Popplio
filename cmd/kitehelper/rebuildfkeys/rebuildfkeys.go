package rebuildfkeys

import (
	"context"
	"fmt"
	"kitehelper/common"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ctx = context.Background()
	sp  *common.SandboxPool
)

type TableRef struct {
	ConstraintName    string `db:"constraint_name"`
	ForeignTableName  string `db:"foreign_table_name"`
	TableName         string `db:"table_name"`
	ColumnName        string `db:"column_name"`
	ForeignColumnName string `db:"foreign_column_name"`
}

func getAllTableRefs() ([]TableRef, error) {
	rows, err := sp.Query(
		ctx,
		`
SELECT 
  c.conname AS constraint_name,
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
WHERE c.contype = 'f'`,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query table refs: %w", err)
	}

	keys, err := pgx.CollectRows(rows, pgx.RowToStructByName[TableRef])

	if err != nil {
		return nil, fmt.Errorf("failed to collect rows: %w", err)
	}

	return keys, nil
}

func RebuildFKeys(progname string, args []string) {
	_pool, err := pgxpool.New(ctx, "postgres:///infinity")

	if err != nil {
		panic(err)
	}

	sp = common.NewSandboxPool(_pool)

	// Get a list of all tables
	tables, err := getAllTableRefs()

	if err != nil {
		panic(err)
	}

	sp.AllowCommit = true

	spTx, err := sp.Begin(ctx)

	if err != nil {
		panic(err)
	}

	defer spTx.Rollback(ctx)

	for i, table := range tables {
		fmt.Printf("[%d/%d] %s (%s/%s -> %s/%s)\n", i+1, len(tables), table.ConstraintName, table.TableName, table.ColumnName, table.ForeignTableName, table.ForeignColumnName)

		err = spTx.Exec(ctx, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", table.TableName, table.ConstraintName))

		if err != nil {
			fmt.Println("failed to drop constraint:", err)
			continue
		}

		err = spTx.Exec(ctx, fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON UPDATE CASCADE ON DELETE CASCADE", table.TableName, table.ConstraintName, table.ColumnName, table.ForeignTableName, table.ForeignColumnName))

		if err != nil {
			fmt.Println("failed to add constraint:", err)
			continue
		}
	}

	err = spTx.Commit(ctx)

	if err != nil {
		panic(err)
	}
}
