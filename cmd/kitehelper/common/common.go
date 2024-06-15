package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
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

// Creates a new discord token, returning the session once it has recieved READY event
func NewDiscordSession(token string) (*discordgo.Session, error) {
	fmt.Println("[NewDiscordSession] Creating new discord session and waiting for READY event...")
	sess, err := discordgo.New("Bot " + token)

	if err != nil {
		return nil, err
	}

	var readyChan = make(chan struct{})
	sess.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("[NewDiscordSession] Discord session ready")
		close(readyChan)
	})

	sess.Identify.Intents = discordgo.IntentsAll

	err = sess.Open()

	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-readyChan:
			fmt.Println("Waiting 30 seconds for session to be populated...")
			time.Sleep(30 * time.Second) // Give some more time for the session to be ready
			return sess, nil
		case <-time.After(600 * time.Second):
			return nil, errors.New("timed out waiting for READY event")
		}
	}
}
