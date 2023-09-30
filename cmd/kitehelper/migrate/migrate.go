package migrate

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_pool         *pgxpool.Pool
	Ctx           = context.Background()
	migrationList []Migration
	//discordSess *discordgo.Session

	StatusBoldBlue   = color.New(color.Bold, color.FgBlue).PrintlnFunc()
	StatusGood       = color.New(color.Bold, color.FgCyan).PrintlnFunc()
	StatusBoldYellow = color.New(color.Bold, color.FgYellow).PrintlnFunc()
)

type SandboxPool struct {
	currentMigration *Migration
	allowCommit      bool
	pool             *pgxpool.Pool
}

func (s *SandboxPool) Log(typ string, args ...interface{}) {
	fmt.Println("sandboxPool - ", typ+":", args)
}

func (s *SandboxPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	s.Log("QueryRow", sql, "with arguments:", args)
	return s.pool.QueryRow(ctx, sql, args...)
}

func (s *SandboxPool) Exec(ctx context.Context, sql string, args ...interface{}) error {
	s.Log("Exec", sql, "with arguments:", args)

	if os.Getenv("COMMIT") != "" && s.allowCommit {
		_, err := s.pool.Exec(ctx, sql, args...)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SandboxPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	s.Log("Query", sql, "with arguments:", args)
	return s.pool.Query(ctx, sql, args...)
}

func (s *SandboxPool) Transaction(ctx context.Context, calls []func(tx pgx.Tx)) error {
	if !s.allowCommit {
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

func AddMigrations(mig []Migration) {
	migrationList = append(migrationList, mig...)
}

type Migration struct {
	ID          string // Mandatory
	Name        string // Mandatory
	HasMigrated func(pool *SandboxPool) error
	Function    func(pool *SandboxPool)
	Disabled    bool
}

func Migrate(progname string, args []string) {
	var err error
	_pool, err = pgxpool.New(Ctx, "postgres:///infinity")

	if err != nil {
		panic(err)
	}

	/*if os.Getenv("DISCORD_TOKEN") == "" {
		panic("DISCORD_TOKEN not set. Please set it to a discord token to allow some migration steps to run.")
	}

	discordSess, err = discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))

	if err != nil {
		panic(err)
	}*/

	sandboxPoolWrapper := &SandboxPool{pool: _pool} // Used to prevent migrations from being able to commit whatever they want

	for i, mig := range migrationList {
		if mig.Disabled {
			continue
		}

		if mig.ID == "" {
			panic("Migration #" + strconv.Itoa(i) + " is missing mandatory field \"id\"")
		}

		if mig.Name == "" {
			panic("Migration #" + strconv.Itoa(i) + " is missing mandatory field \"name\"")
		}

		if mig.Function == nil {
			panic("Migration #" + strconv.Itoa(i) + " is missing mandatory field \"function\"")
		}

		if mig.HasMigrated == nil {
			panic("Migration #" + strconv.Itoa(i) + " is missing mandatory field \"hasMigrated\"")
		}

		StatusBoldBlue("Running migration:", mig.Name, "["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(migrationList))+"]", "("+mig.ID+")")

		sandboxPoolWrapper.currentMigration = &mig
		sandboxPoolWrapper.allowCommit = false

		if err := mig.HasMigrated(sandboxPoolWrapper); err != nil {
			StatusGood("Already migrated, nothing to do here...")
			continue
		}

		sandboxPoolWrapper.allowCommit = true
		mig.Function(sandboxPoolWrapper)
	}
}
