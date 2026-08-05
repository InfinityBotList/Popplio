package bot

import (
	"fmt"
	"strconv"
	"strings"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/tasks"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// Staff tooling that acts on the platform: the staff roster, a bot's Discord
// roles, forcing a dovewing refresh, and the catalogue of RPC actions.
//
// The RPC actions themselves are driven from interactions.go, which builds a
// modal per method; this file only lists what exists.

func cmdStaff() *Command {
	return &Command{
		Name:        "staff",
		Category:    "Staff",
		Description: "Staff related commands",
		Run: func(c *Ctx) error {
			return c.Say("Some available options are ``staff list``, ``staff guildlist``, ``staff_guildleave``, ``staff_stats``")
		},
		Subcommands: []*Command{
			{
				Name:        "list",
				Category:    "Staff",
				Description: "Get the list of staff members",
				// DISABLED upstream: the body is commented out and the command
				// always errors. Reproduced as a disabled stub.
				Run: nil,
			},
			{
				Name:        "guildlist",
				Category:    "Staff",
				Description: "Get the list of guilds the bot is in",
				Run: func(c *Ctx) error {
					var sb strings.Builder

					dclient.Get().Caches().GuildsForEach(func(guild discord.Guild) {
						fmt.Fprintf(&sb, "%s (%s)\n", guild.Name, guild.ID)
					})

					if sb.Len() == 0 {
						return c.Say("Unknown")
					}

					return c.Say(sb.String())
				},
			},
			{
				Name:        "guildleave",
				Category:    "Staff",
				Description: "Leave a guild",
				Checks:      []Check{staffServer},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "guild", Description: "The guild ID to leave", Required: true},
				},
				Run: func(c *Ctx) error {
					if err := requirePerm(c, perms.StaffAdministrator); err != nil {
						return err
					}

					guildID, err := snowflake.Parse(c.Option("guild", 0))

					if err != nil {
						return err
					}

					if err := dclient.Get().Rest().LeaveGuild(guildID); err != nil {
						return err
					}

					return c.Ok("Removed guild")
				},
			},
			{
				Name:        "stats",
				Category:    "Staff",
				Description: "Get the stats of a staff member",
				Checks:      []Check{staffServer},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionUser{Name: "user", Description: "The staff member you are looking for?"},
				},
				Run: func(c *Ctx) error {
					userID := c.Option("user", 0)

					if userID == "" {
						userID = c.Author.ID.String()
					}

					userID = strings.Trim(userID, "<@!>")

					rows, err := state.Pool.Query(c.Context, "SELECT method, COUNT(*) FROM rpc_logs WHERE user_id = $1 GROUP BY method", userID)

					if err != nil {
						return err
					}

					defer rows.Close()

					fields := []discord.EmbedField{
						{Name: "Username", Value: c.Author.Username, Inline: impls.InlineTrue()},
						{Name: "User ID", Value: userID, Inline: impls.InlineTrue()},
					}

					for rows.Next() {
						var (
							method string
							count  int64
						)

						if err := rows.Scan(&method, &count); err != nil {
							return err
						}

						if count == 0 {
							continue
						}

						fields = append(fields, discord.EmbedField{
							Name:   method,
							Value:  strconv.FormatInt(count, 10),
							Inline: impls.InlineTrue(),
						})
					}

					if err := rows.Err(); err != nil {
						return err
					}

					return c.Send(discord.MessageCreate{
						Embeds: []discord.Embed{{
							Title:  "Staff Stats",
							Fields: fields,
							Color:  impls.ColourGreen,
						}},
					})
				},
			},
		},
	}
}

func cmdGetBotRoles() *Command {
	return &Command{
		Name:        "getbotroles",
		Category:    "Bot Owner",
		Description: "Get the roles you are entitled to for the bots you own",
		Checks:      []Check{mainServer},
		Run: func(c *Ctx) error {
			owned, err := impls.GetOwnedBy(c.Context, c.Author.ID.String())

			if err != nil {
				return err
			}

			if len(owned) == 0 {
				return fmt.Errorf("You are not the owner/additional owner of any bots")
			}

			var hasApproved, hasCertified bool

			for _, entity := range owned {
				if entity.TargetType != types.TargetTypeBot {
					continue
				}

				switch entity.EntityState {
				case "approved":
					hasApproved = true
				case "certified":
					hasCertified = true
				}
			}

			if !hasApproved && !hasCertified {
				return fmt.Errorf("You are not the owner/additional owner of any approved or certified bots")
			}

			if !impls.MemberOnGuild(state.Config.Servers.Main, c.Author.ID) {
				return fmt.Errorf("You are not in the server")
			}

			err = impls.AddRole(state.Config.Servers.Main, c.Author.ID, state.Config.Roles.BotDeveloper, "Autorole due to bots owned")

			if err != nil {
				return err
			}

			if hasCertified {
				err = impls.AddRole(state.Config.Servers.Main, c.Author.ID, state.Config.Roles.CertifiedDeveloper, "Autorole due to bots owned")

				if err != nil {
					return err
				}

				return c.Ok("You are the owner/additional owner of a certified bot! Giving you certified role")
			}

			err = impls.RemoveRole(state.Config.Servers.Main, c.Author.ID, state.Config.Roles.CertifiedDeveloper, "Autorole due to bots owned")

			if err != nil {
				return err
			}

			return c.Ok("You are the owner/additional owner of an approved bot! Giving you approved role")
		},
	}
}

func cmdRefresh() *Command {
	return &Command{
		Name:        "refresh",
		Category:    "Leaderboard",
		Description: "Force refresh the top reviewer roles",
		Checks:      []Check{staffServer},
		Run: func(c *Ctx) error {
			if err := requirePerm(c, perms.StaffAdministrator); err != nil {
				return err
			}

			// The command uses a limit of 3; the weekly task uses 0. See
			// CONFORMANCE.md.
			stats, err := tasks.QueryTopReviewers(c.Context, 3)

			if err != nil {
				return err
			}

			if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Main); !ok {
				return fmt.Errorf("Failed to get guild")
			}

			if err := tasks.SyncTopReviewerRoles(c.Context, stats); err != nil {
				return err
			}

			return c.Ok("**Force Refresh**\nSynced Top Reviewers!")
		},
	}
}

func cmdRPCList() *Command {
	return &Command{
		Name:        "rpclist",
		Category:    "RPC",
		Description: "Lists all available RPC actions",
		Run: func(c *Ctx) error {
			fields := make([]discord.EmbedField, 0, len(types.RPCMethodVariants))

			for _, name := range types.RPCMethodVariants {
				variant, err := types.EmptyRPCMethod(name)

				if err != nil {
					return err
				}

				var sb strings.Builder

				fmt.Fprintf(&sb, "%s\n", variant.Description())

				for _, field := range variant.Fields() {
					fmt.Fprintf(&sb, "\n**%s**: %s", field.Label, field.Placeholder)
				}

				fields = append(fields, discord.EmbedField{
					Name:  variant.Label(),
					Value: sb.String(),
				})
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:  "RPC Actions",
					Fields: fields,
					Color:  impls.ColourGreen,
				}},
			})
		},
	}
}
