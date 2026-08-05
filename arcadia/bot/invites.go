package bot

import (
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// Resolving a bot invite: what the list has on record, and what Discord would
// actually hand out.

func cmdInviteDB() *Command {
	return &Command{
		Name:        "invite_db",
		Category:    "Testing",
		Description: "Gets the invite to a bot",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "bot", Description: "The invite to the bot", Required: true},
		},
		Run: func(c *Ctx) error {
			var invite string

			err := state.Pool.QueryRow(c.Context,
				"SELECT invite FROM bots WHERE bot_id = $1 ORDER BY created_at DESC LIMIT 1",
				c.Option("bot", 0)).Scan(&invite)

			if err != nil {
				return err
			}

			return c.Say(fmt.Sprintf("Invite: %s", invite))
		},
	}
}

func cmdInvite() *Command {
	return &Command{
		Name:        "invite",
		Category:    "Testing",
		Description: "Gets a safe invite to a bot",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "bot", Description: "The invite to the bot", Required: true},
		},
		Run: func(c *Ctx) error {
			var clientID string

			err := state.Pool.QueryRow(c.Context, "SELECT client_id FROM bots WHERE bot_id = $1", c.Option("bot", 0)).Scan(&clientID)

			if err != nil {
				return err
			}

			return c.Say(fmt.Sprintf(
				"https://discord.com/api/v10/oauth2/authorize?client_id=%s&permissions=0&scope=bot%%20applications.commands&guild_id=%s",
				clientID, state.Config.Servers.Testing))
		},
	}
}
