package common

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SandboxTx struct {
	sp        *SandboxPool
	tx        pgx.Tx
	committed bool
}

func (s *SandboxTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	s.sp.Log("QueryRow", sql, "with arguments:", args)
	return s.tx.QueryRow(ctx, sql, args...)
}

func (s *SandboxTx) Exec(ctx context.Context, sql string, args ...interface{}) error {
	s.sp.Log("Exec", sql, "with arguments:", args)

	if os.Getenv("NO_COMMIT") == "" && s.sp.AllowCommit {
		s.sp.Log("Exec", "Allowing commit")
		_, err := s.tx.Exec(ctx, sql, args...)

		if err != nil {
			return err
		}
	} else {
		s.sp.Log("Exec", "Denying exec silently")
	}

	return nil
}

func (s *SandboxTx) Rollback(ctx context.Context) error {
	if s.committed {
		s.sp.Log("Rollback", "Already committed, not rolling back")
	} else {
		s.sp.Log("Rollback", "Rolling back")
	}
	return s.tx.Rollback(ctx)
}

func (s *SandboxTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	s.sp.Log("Query", sql, "with arguments:", args)
	return s.tx.Query(ctx, sql, args...)
}

func (s *SandboxTx) Commit(ctx context.Context) error {
	if os.Getenv("NO_COMMIT") == "" && s.sp.AllowCommit {
		s.sp.Log("Commit", "Allowing commit")
		s.committed = true
		return s.tx.Commit(ctx)
	}

	s.sp.Log("Commit", "Denying commit silently")
	return nil
}

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

func (s *SandboxPool) Begin(ctx context.Context) (*SandboxTx, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	s.Log("Begin", "Beginning transaction at time =", time.Now())

	return &SandboxTx{
		sp: s,
		tx: tx,
	}, nil
}
