package common

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
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
func NewDiscordSession(token string) (bot.Client, error) {
	fmt.Println("[NewDiscordSession] Creating new discord session and waiting for READY event...")

	var readyChan = make(chan struct{})
	sess, err := disgo.New(token, bot.WithShardManagerConfigOpts(
		sharding.WithShardIDs(0, 1),
		sharding.WithShardCount(2),
		sharding.WithAutoScaling(true),
		sharding.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildPresences, gateway.IntentGuildMembers),
			gateway.WithCompress(true),
		),
	),
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds|cache.FlagMembers|cache.FlagPresences),
		),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnGuildReady: func(event *events.GuildReady) {
				fmt.Println("[NewDiscordSession] Guild ready:", event.Guild.ID.String())
			},
			OnGuildsReady: func(event *events.GuildsReady) {
				fmt.Println("[NewDiscordSession] Discord Session ready")
				close(readyChan)
			},
		}),
	)

	if err != nil {
		return nil, err
	}

	if err = sess.OpenShardManager(context.Background()); err != nil {
		slog.Error("error while connecting to gateway", slog.Any("err", err))
		return sess, err
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
