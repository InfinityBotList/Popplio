package migrate

import (
	"context"
	"kitehelper/common"
	"strconv"

	"github.com/fatih/color"
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

func AddMigrations(mig []Migration) {
	migrationList = append(migrationList, mig...)
}

type Migration struct {
	ID          string // Mandatory
	Name        string // Mandatory
	HasMigrated func(pool *common.SandboxPool) error
	Function    func(pool *common.SandboxPool)
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

	sandboxPoolWrapper := common.NewSandboxPool(_pool) // Used to prevent migrations from being able to commit whatever they want

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

		sandboxPoolWrapper.AllowCommit = false

		if err := mig.HasMigrated(sandboxPoolWrapper); err != nil {
			StatusGood("Already migrated, nothing to do here...")
			continue
		}

		sandboxPoolWrapper.AllowCommit = true
		mig.Function(sandboxPoolWrapper)
	}
}
