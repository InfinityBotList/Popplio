package assets

import (
	"fmt"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

type TableRef struct {
	ForeignTable      string `db:"foreign_table_name"`
	TableName         string `db:"table_name"`
	ColumnName        string `db:"column_name"`
	ForeignColumnName string `db:"foreign_column_name"`
}

func getAllTableRefs() ([]TableRef, error) {
	rows, err := state.Pool.Query(
		state.Context,
		`
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
WHERE c.contype = 'f'`,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query table refs: %w", err)
	}

	keys, err := pgx.CollectRows(rows, pgx.RowToStructByName[TableRef])

	if err != nil {
		return nil, fmt.Errorf("failed to collect rows: %w", err)
	}

	keys = append(keys, TableRef{
		ForeignTable:      "users",
		TableName:         "users",
		ColumnName:        "user_id",
		ForeignColumnName: "user_id",
	})

	return keys, nil
}
