package main

import (
	"fmt"
	_ "kitehelper/icb/icb_migrations"
	"kitehelper/migrate"
	"kitehelper/rebuildfkeys"
	"kitehelper/tests"
	"kitehelper/validatetable"
	"os"
	"runtime/debug"
	"slices"
)

var GitCommit string

func init() {
	// Use runtime/debug vcs.revision to get the git commit hash
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				GitCommit = setting.Value
			}
		}
	}

	if GitCommit == "" {
		GitCommit = "unknown"
	}
}

type command struct {
	Func func(progname string, args []string)
	Help string
}

var cmds = map[string]command{
	"test": {
		Func: tests.Tester,
		Help: "Run tests [Set NO_INTERACTION environment variable to disable all input interaction]",
	},
	"migrate": {
		Func: migrate.Migrate,
		Help: "Run custom migrations",
	},
	"validate-table": {
		Func: validatetable.ValidateTable,
		Help: "Validate a table",
	},
	"rebuild-fkeys": {
		Func: rebuildfkeys.RebuildFKeys,
		Help: "Rebuild foreign keys",
	},
}

func cmdList() {
	fmt.Println("Commands:")
	for k, cmd := range cmds {
		fmt.Println(k+":", cmd.Help)
	}
}

func main() {
	progname := os.Args[0]
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Printf("usage: %s <command> [args]\n\n", progname)
		cmdList()
		os.Exit(1)
	}

	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		fmt.Printf("usage: %s <command> [args]\n\n", progname)
		cmdList()
		os.Exit(0)
	}

	cmd, ok := cmds[args[0]]
	if !ok {
		fmt.Printf("unknown command: %s\n\n", args[0])
		cmdList()
		os.Exit(1)
	}

	fmt.Printf("Kitehelper (commit: %s)\n", GitCommit)

	cmd.Func(progname, args[1:])
}
