package tests

import (
	"fmt"
	"kitehelper/common"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	statusGood       = color.New(color.Bold, color.FgCyan).PrintlnFunc()
	statusSuccess    = color.New(color.Bold, color.FgGreen).PrintlnFunc()
	statusBoldYellow = color.New(color.Bold, color.FgYellow).PrintlnFunc()
	statusBoldErr    = color.New(color.Bold, color.FgRed).PrintlnFunc()
	statusBoldBlue   = color.New(color.Bold, color.FgBlue).PrintlnFunc()
	statusBoldBlueS  = color.New(color.Bold, color.FgBlue).SprintFunc()
)

type test struct {
	name         string
	cmd          []string
	cwd          string
	ignoreErrors string
	customTest   string
}

type testset struct {
	Tests []test
}

func (ts testset) Run() {
	// Unpack all custom tests
	customTestsFsd, err := customTests.ReadDir("custom")

	if err != nil {
		panic(err)
	}

	// Write to temp folder
	tmpFolder, err := os.MkdirTemp("", "tmp")

	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(tmpFolder)

	for _, t := range customTestsFsd {
		testFile, err := customTests.ReadFile("custom/" + t.Name())

		if err != nil {
			panic(err)
		}

		err = os.WriteFile(tmpFolder+"/"+t.Name(), testFile, 0644)

		if err != nil {
			panic(err)
		}
	}

	failed := []test{}
	success := []test{}
	outputs := []string{}

	os.Setenv("PATH", os.Getenv("PATH")+":.")
	os.Setenv("FORCE_COLOR", "1")

	for i, t := range ts.Tests {
		err := os.Chdir(common.GetRepoRoot())
		if err != nil {
			panic(err)
		}
		if t.cwd != "" {
			os.Chdir(t.cwd)
		}

		currDir, err := os.Getwd()

		if err != nil {
			panic(err)
		}

		statusGood(t.name, "["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(ts.Tests))+"] (in", currDir+")")

		var cmdErr error
		var cmdOut []byte

		if t.customTest != "" {
			t.cmd = append(t.cmd, tmpFolder+"/"+t.customTest)
		}

		// Run test here
		if len(t.cmd) > 0 {
			cmd := exec.Command(t.cmd[0], t.cmd[1:]...)

			cmd.Env = os.Environ()

			cmdOut, cmdErr = cmd.CombinedOutput()

			if os.Getenv("DEBUG") == "1" {
				fmt.Println(string(cmdOut))
			}
		}

		outputs = append(outputs, string(cmdOut))

		if cmdErr != nil {
			if t.ignoreErrors != "" {
				statusBoldErr("Test failed, but ignoring error:", t.ignoreErrors)
				time.Sleep(1 * time.Second)
				success = append(success, t)
				continue
			}
			failed = append(failed, t)

			// Print last 10 lines of output
			lines := strings.Split(string(cmdOut), "\n")

			if len(lines) > 10 {
				lines = lines[len(lines)-10:]
			}

			statusBoldErr("Showing last 10 lines of output:")
			for _, line := range lines {
				fmt.Println(line)
			}

			statusBoldYellow("Test", t.name, "has failed!")

			var inp string
			if os.Getenv("NO_INTERACTION") == "" {
				inp = common.AskInput("Continue (y/N): ")
			}
			if inp == "y" || inp == "Y" {
				continue
			} else {
				fmt.Println(string(cmdOut))
				statusBoldYellow("Output of test", t.name, "is above.")
				break
			}
		} else {
			success = append(success, t)
		}
	}

	fmt.Println("")

	if len(success) > 0 {
		fmt.Println("")
		statusSuccess("Successful tests:")
		for _, t := range success {
			statusSuccess(t.name, "["+strings.Join(t.cmd, " ")+"]")
		}
	}

	if len(failed) > 0 {
		fmt.Println("")
		statusBoldErr("Failed tests:")
		for _, t := range failed {
			statusBoldErr(t.name, "["+strings.Join(t.cmd, " ")+"]")
		}
	}

	if os.Getenv("NO_INTERACTION") == "" {
		statusBoldBlue("List of all tests:")
		for i, t := range ts.Tests {
			fmt.Println(strconv.Itoa(i+1) + ": " + t.name + " [" + strings.Join(t.cmd, " ") + "]")
		}

		for {
			userOut := common.AskInput(statusBoldBlueS("Which test number would you like to see the output of (hit ENTER to exit): "))

			if userOut != "" {
				num, err := strconv.Atoi(userOut)

				if err != nil {
					statusBoldErr("Invalid input")
					continue
				}

				common.PageOutput(outputs[num-1])
			} else {
				break
			}
		}
	}

	if len(failed) > 0 {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func Tester(progname string, args []string) {
	testList.Run()
}
