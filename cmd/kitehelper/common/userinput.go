package common

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	StatusBoldBlue   = color.New(color.Bold, color.FgBlue).PrintlnFunc()
	StatusGood       = color.New(color.Bold, color.FgCyan).PrintlnFunc()
	StatusBoldYellow = color.New(color.Bold, color.FgYellow).PrintlnFunc()
)

func UserInputBoolean(prompt string) bool {
	for {
		var input string
		StatusBoldYellow(prompt + " (y/n): ")
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

		StatusBoldYellow("Invalid input, please try again.")
	}
}

func UserInput(prompt string) string {
	for {
		var input string
		StatusBoldYellow(prompt + ": ")
		_, err := fmt.Scanln(&input)

		if err != nil {
			panic(err)
		}

		if input == "" {
			StatusBoldYellow("Invalid input, please try again.")
			continue
		}

		return input
	}
}
