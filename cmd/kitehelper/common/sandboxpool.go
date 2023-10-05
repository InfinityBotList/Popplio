package common

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SandboxPool struct {
	AllowCommit bool
	pool        *pgxpool.Pool
}

func NewSandboxPool(pool *pgxpool.Pool) *SandboxPool {
	return &SandboxPool{
		AllowCommit: false,
		pool:        pool,
	}
}

func (s *SandboxPool) Log(typ string, args ...interface{}) {
	if os.Getenv("NO_LOGS") == "" {
		fmt.Println("sandboxPool - ", typ+":", args)
	}
}

func (s *SandboxPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	s.Log("QueryRow", sql, "with arguments:", args)
	return s.pool.QueryRow(ctx, sql, args...)
}

func (s *SandboxPool) Exec(ctx context.Context, sql string, args ...interface{}) error {
	s.Log("Exec", sql, "with arguments:", args)

	if os.Getenv("NO_COMMIT") == "" && s.AllowCommit {
		s.Log("Exec", "Allowing commit")
		_, err := s.pool.Exec(ctx, sql, args...)

		if err != nil {
			return err
		}
	} else {
		s.Log("Exec", "Denying exec silently")
	}

	return nil
}

func (s *SandboxPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	s.Log("Query", sql, "with arguments:", args)
	return s.pool.Query(ctx, sql, args...)
}

func (s *SandboxPool) Transaction(ctx context.Context, calls []func(tx pgx.Tx)) error {
	if os.Getenv("NO_COMMIT") == "" && s.AllowCommit {
		panic("creating a transaction is not allowed in this scope")
	}

	s.Log("Transaction", "with", strconv.Itoa(len(calls)), "calls started")
	tx, err := s.pool.Begin(ctx)
	defer tx.Rollback(ctx)

	if err != nil {
		return err
	}

	for _, call := range calls {
		call(tx)
	}

	s.Log("Transaction", "with", strconv.Itoa(len(calls)), "calls committed")

	err = tx.Commit(ctx)

	if err != nil {
		return err
	}

	return nil
}
