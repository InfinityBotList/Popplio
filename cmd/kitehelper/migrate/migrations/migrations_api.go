package migrations

import (
	"fmt"
	"kitehelper/migrate"
)

var ctx = migrate.Ctx

func tableExists(pool *migrate.SandboxPool, name string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", name).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func colExists(pool *migrate.SandboxPool, table, col string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)", table, col).Scan(&exists)

	if err != nil {
		panic(err)
	}

	return exists
}

func userInputBoolean(prompt string) bool {
	for {
		var input string
		migrate.StatusBoldYellow(prompt + " (y/n): ")
		_, err := fmt.Scanln(&input)

		if err != nil {
			panic(err)
		}

		if input == "y" || input == "Y" {
			return true
		}

		if input == "n" || input == "N" {
			return false
		}

		migrate.StatusBoldYellow("Invalid input, please try again.")
	}
}

func userInput(prompt string) string {
	for {
		var input string
		migrate.StatusBoldYellow(prompt + ": ")
		_, err := fmt.Scanln(&input)

		if err != nil {
			panic(err)
		}

		if input == "" {
			migrate.StatusBoldYellow("Invalid input, please try again.")
			continue
		}

		return input
	}
}
