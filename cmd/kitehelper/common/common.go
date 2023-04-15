package common

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
)

var (
	Fatal = func(a ...any) {
		color.New(color.FgRed, color.Bold).PrintlnFunc()(a...)
		os.Exit(1)
	}
)

func GetRepoRoot() string {
	// Use git rev-parse --show-toplevel to get the root of the repo
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()

	if err != nil {
		panic(err)
	}

	return strings.ReplaceAll(string(out), "\n", "")
}

func AskInput(question string) string {
	fmt.Print(question)
	scanner := bufio.NewScanner(os.Stdin)

	var opt string

	for scanner.Scan() {
		opt = scanner.Text()
		break
	}

	if scanner.Err() != nil {
		// Handle error.
		Fatal(scanner.Err())
	}

	return opt
}

func PageOutput(text string) {
	// Custom pager using less -r
	text = "\n" + text

	cmd := exec.Command("less", "-r")
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = os.Stdout
	cmd.Run()
}
