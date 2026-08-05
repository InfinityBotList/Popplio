package bot

import (
	"fmt"
	"strings"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// The commands that explain the bot and the review process rather than doing
// anything: help, explainme and staffguide. The explainme texts themselves are
// frozen verbatim in explain.go.

func cmdHelp() *Command {
	return &Command{
		Name:        "help",
		Category:    "Help",
		Description: "Shows help for a command or lists all commands",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "command", Description: "Command to show help for"},
		},
		Run: func(c *Ctx) error {
			name := c.Option("command", 0)

			if name != "" {
				cmd, ok := registry[strings.ToLower(name)]

				if !ok {
					return c.Fail(fmt.Sprintf("No such command: %s", name))
				}

				var sb strings.Builder

				fmt.Fprintf(&sb, "**%s%s**\n%s", prefix(), cmd.Name, cmd.Description)

				for _, sub := range cmd.Subcommands {
					fmt.Fprintf(&sb, "\n- ``%s %s``: %s", cmd.Name, sub.Name, sub.Description)
				}

				return c.Say(sb.String())
			}

			var sb strings.Builder

			sb.WriteString("**Commands**\n")

			for _, category := range helpCategories() {
				fmt.Fprintf(&sb, "\n__%s__\n", category)

				for _, cmd := range ordered {
					if cmd.Category != category {
						continue
					}

					fmt.Fprintf(&sb, "``%s%s``: %s\n", prefix(), cmd.Name, cmd.Description)
				}
			}

			return c.Say(sb.String())
		},
	}
}

func cmdExplainMe() *Command {
	return &Command{
		Name:        "explainme",
		Category:    "Explain",
		Description: "An explaination of how the bot works",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "command",
				Description: "Command",
				Choices: []discord.ApplicationCommandOptionChoiceString{
					{Name: "Claim", Value: "Claim"},
					{Name: "Testing", Value: "Testing"},
					{Name: "Approve/Deny", Value: "ApproveDeny"},
					{Name: "Tips", Value: "Tips"},
				},
			},
		},
		Run: func(c *Ctx) error {
			return c.Say(explainText(c.Option("command", 0)))
		},
	}
}

func cmdStaffGuide() *Command {
	return &Command{
		Name:        "staffguide",
		Category:    "Testing",
		Description: "Sends the staff guide link",
		Run: func(c *Ctx) error {
			return c.Say(fmt.Sprintf(
				"The staff guide can be found at %s/staff/guide. Please **do not** bookmark this page as the URL may change in the future",
				state.Config.Sites.Frontend.Parse()))
		},
	}
}
