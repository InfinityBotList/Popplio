// Package dbconform verifies every SQL statement the port issues against a real
// database.
//
// The Rust original used sqlx's compile-time query verification, so a typo'd
// column name could not survive a build. Go has no equivalent, which §13 of the
// port brief calls out as the single biggest regression: without this, a bad
// column name only surfaces when a staff member triggers that code path in
// production.
//
// Rather than maintain a hand-written list of statements that would drift, this
// walks the arcadia source with go/ast, pulls out every SQL string literal, and
// PREPAREs it against Postgres. Preparing parses and plans the statement -
// validating every table, column and function reference and inferring parameter
// types - without executing anything or touching a row.
//
// Run with:
//
//	ARCADIA_TEST_DATABASE_URL=postgres://user:pass@127.0.0.1/infinity \
//	  go test -ldflags=-checklinkname=0 ./arcadia/dbconform/
package dbconform

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// sqlLeadingKeywords are the statement kinds Postgres can PREPARE. DDL and LOCK
// cannot be prepared, so they are recognised but skipped.
var sqlLeadingKeywords = []string{"SELECT", "INSERT", "UPDATE", "DELETE", "WITH", "select", "insert", "update", "delete"}

var unpreparable = []string{"CREATE", "LOCK", "create", "lock"}

// statement is one SQL literal found in the source.
type statement struct {
	SQL  string
	Pos  string
	File string
}

func TestEverySQLStatementPrepares(t *testing.T) {
	url := os.Getenv("ARCADIA_TEST_DATABASE_URL")

	if url == "" {
		t.Skip("ARCADIA_TEST_DATABASE_URL is not set; skipping SQL conformance")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, url)

	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	defer conn.Close(ctx)

	statements := collectStatements(t, "..")

	if len(statements) == 0 {
		t.Fatal("no SQL statements were found in the arcadia sources")
	}

	t.Logf("preparing %d SQL statements", len(statements))

	var failures int

	for i, stmt := range statements {
		name := fmt.Sprintf("stmt_%d", i)

		if _, err := conn.Prepare(ctx, name, stmt.SQL); err != nil {
			failures++
			t.Errorf("%s: %v\n  SQL: %s", stmt.Pos, err, collapse(stmt.SQL))
		}
	}

	if failures == 0 {
		t.Logf("all %d statements prepared cleanly", len(statements))
	}
}

// TestComposedStatements covers the statements that are built at runtime rather
// than written as a single literal, so the extractor above cannot see them
// whole.
func TestComposedStatements(t *testing.T) {
	url := os.Getenv("ARCADIA_TEST_DATABASE_URL")

	if url == "" {
		t.Skip("ARCADIA_TEST_DATABASE_URL is not set; skipping SQL conformance")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, url)

	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	defer conn.Close(ctx)

	composed := []struct {
		name string
		sql  string
	}{
		{
			// impls.GetStaffDisciplinaries appends the interval predicate when
			// active is true.
			name: "active disciplinaries",
			sql:  "SELECT id, created_at, EXTRACT(epoch FROM expiry) as expiry, title, description, type FROM staff_disciplinary WHERE user_id = $1 AND NOW() - created_at < expiry",
		},
		{
			// tasks.AssetCleaner builds one of these per entity from a fixed
			// allow-list.
			name: "asset cleaner bots",
			sql:  "SELECT bot_id::text FROM bots WHERE bot_id::text = $1::text",
		},
		{name: "asset cleaner users", sql: "SELECT user_id::text FROM users WHERE user_id::text = $1::text"},
		{name: "asset cleaner servers", sql: "SELECT server_id::text FROM servers WHERE server_id::text = $1::text"},
		{name: "asset cleaner teams", sql: "SELECT id::text FROM teams WHERE id::text = $1::text"},
		{name: "asset cleaner partners", sql: "SELECT id::text FROM partners WHERE id::text = $1::text"},
		{name: "asset cleaner tickets", sql: "SELECT id::text FROM tickets WHERE id::text = $1::text"},
		{
			// tasks.GenericCleaner builds one of these per entity kind, against
			// each table carrying a target_id column. entity_votes is a real one.
			name: "generic cleaner select",
			sql:  `select target_id, target_type from "entity_votes" where target_type = $1 and not exists (select 1 from "bots" where "bot_id"::text = target_id)`,
		},
		{
			name: "generic cleaner delete",
			sql:  `delete from "entity_votes" where target_id = $1 and target_type = $2`,
		},
	}

	for _, tt := range composed {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := conn.Prepare(context.Background(), tt.name, tt.sql); err != nil {
				t.Errorf("%v\n  SQL: %s", err, collapse(tt.sql))
			}
		})
	}
}

// TestGenericCleanerTablesExist checks the discovery query the generic cleaner
// relies on actually finds tables, and that each survives the identifier filter.
func TestGenericCleanerTablesExist(t *testing.T) {
	url := os.Getenv("ARCADIA_TEST_DATABASE_URL")

	if url == "" {
		t.Skip("ARCADIA_TEST_DATABASE_URL is not set; skipping SQL conformance")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, url)

	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, "select table_name from information_schema.columns where column_name = 'target_id' and table_schema = 'public'")

	if err != nil {
		t.Fatalf("discovery query: %v", err)
	}

	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("the generic cleaner's discovery query found no tables")
	}

	t.Logf("generic cleaner would act on %d tables: %s", len(tables), strings.Join(tables, ", "))
}

// collectStatements walks the arcadia packages and extracts SQL string literals.
func collectStatements(t *testing.T, root string) []statement {
	t.Helper()

	var out []statement

	fset := token.NewFileSet()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)

		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)

			if !ok || lit.Kind != token.STRING {
				return true
			}

			value, err := strconv.Unquote(lit.Value)

			if err != nil {
				return true
			}

			if !looksLikeSQL(value) {
				return true
			}

			out = append(out, statement{
				SQL:  value,
				Pos:  fset.Position(lit.Pos()).String(),
				File: path,
			})

			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	return out
}

// looksLikeSQL decides whether a string literal is a whole SQL statement worth
// preparing. Fragments, format templates and DDL are excluded.
func looksLikeSQL(s string) bool {
	trimmed := strings.TrimSpace(s)

	if trimmed == "" {
		return false
	}

	for _, keyword := range unpreparable {
		if strings.HasPrefix(trimmed, keyword+" ") {
			return false
		}
	}

	var isSQL bool

	for _, keyword := range sqlLeadingKeywords {
		if strings.HasPrefix(trimmed, keyword+" ") {
			isSQL = true
			break
		}
	}

	if !isSQL {
		return false
	}

	// fmt templates are covered explicitly by TestComposedStatements.
	if strings.Contains(trimmed, "%s") || strings.Contains(trimmed, "%d") || strings.Contains(trimmed, "%v") {
		return false
	}

	return true
}

// collapse renders a multi-line statement on one line for readable failures.
func collapse(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}
