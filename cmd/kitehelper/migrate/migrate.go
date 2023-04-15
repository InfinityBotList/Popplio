package migrate

import (
	"context"
	"strconv"

	"github.com/fatih/color"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pgpool *pgxpool.Pool
	ctx    = context.Background()
	//discordSess *discordgo.Session

	statusBoldBlue = color.New(color.Bold, color.FgBlue).PrintlnFunc()
	statusGood     = color.New(color.Bold, color.FgCyan).PrintlnFunc()
	//statusBoldYellow = color.New(color.Bold, color.FgYellow).PrintlnFunc()
)

type migration struct {
	name     string
	function func()
}

func tableExists(name string) bool {
	var exists bool
	err := pgpool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", name).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func colExists(table, col string) bool {
	var exists bool
	err := pgpool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)", table, col).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func alrMigrated() {
	statusGood("Already migrated, nothing to do here...")
}

func Migrate(progname string, args []string) {
	var err error
	pgpool, err = pgxpool.New(ctx, "postgres:///infinity")

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

	for i, mig := range migs {
		statusBoldBlue("Running migration:", mig.name, "["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(migs))+"]")
		mig.function()
	}
}
