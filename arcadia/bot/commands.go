package bot

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"popplio/arcadia/impls"
	"popplio/arcadia/rpc"
	"popplio/arcadia/tasks"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	perms "github.com/infinitybotlist/kittycat/go"
)

func registerCommands() {
	register(
		cmdRegister(),
		cmdHelp(),
		cmdExplainMe(),
		cmdStaff(),
		cmdInviteDB(),
		cmdInvite(),
		cmdStaffGuide(),
		cmdQueue(),
		cmdClaim(),
		cmdUnclaim(),
		cmdApprove(),
		cmdDeny(),
		cmdAnalytics(),
		cmdInfo(),
		cmdLeaderboard(),
		cmdRefresh(),
		cmdGetBotRoles(),
		cmdRPC(),
		cmdRPCList(),
	)
}

// cmdRegister is poise's owner-only application-command registration helper.
// disgo has no button-driven equivalent, so this syncs the guild commands
// directly.
func cmdRegister() *Command {
	return &Command{
		Name:        "register",
		Category:    "Owner",
		Description: "Register application commands",
		OwnerOnly:   true,
		Run: func(c *Ctx) error {
			if err := SyncCommands(); err != nil {
				return err
			}

			return c.Say("Registered application commands.")
		},
	}
}

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
					return c.Say(fmt.Sprintf("No such command: %s", name))
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

					state.Discord.Caches().GuildsForEach(func(guild discord.Guild) {
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
					if err := requirePerm(c, "arcadia.leave_guilds"); err != nil {
						return err
					}

					guildID, err := snowflake.Parse(c.Option("guild", 0))

					if err != nil {
						return err
					}

					if err := state.Discord.Rest().LeaveGuild(guildID); err != nil {
						return err
					}

					return c.Say("Removed guild")
				},
			},
			{
				Name:        "stats",
				Category:    "Staff",
				Description: "Get the stats of a staff member",
				Checks:      []Check{staffServer},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "user", Description: "The staff member you are looking for?"},
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

func cmdAnalytics() *Command {
	return &Command{
		Name:        "analytics",
		Category:    "Stats",
		Description: "Look at our site analytics!",
		Run: func(c *Ctx) error {
			byType := map[string]int64{}

			rows, err := state.Pool.Query(c.Context, "SELECT type as method, COUNT(*) FROM bots GROUP BY type;")

			if err != nil {
				return err
			}

			for rows.Next() {
				var (
					method string
					count  int64
				)

				if err := rows.Scan(&method, &count); err != nil {
					rows.Close()
					return err
				}

				byType[method] = count
			}

			rows.Close()

			if err := rows.Err(); err != nil {
				return err
			}

			count := func(query string) (int64, error) {
				var n int64
				err := state.Pool.QueryRow(c.Context, query).Scan(&n)
				return n, err
			}

			bots, err := count("SELECT COUNT(*) FROM bots;")

			if err != nil {
				return err
			}

			teams, err := count("SELECT COUNT(*) FROM teams;")

			if err != nil {
				return err
			}

			users, err := count("SELECT COUNT(*) FROM users;")

			if err != nil {
				return err
			}

			guilds, err := count("SELECT COUNT(*) FROM servers;")

			if err != nil {
				return err
			}

			packs, err := count("SELECT COUNT(*) FROM packs;")

			if err != nil {
				return err
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Infinity List Analytics",
					Description: "I hope it's good :eyes:",
					Color:       impls.ColourGreen,
					Fields: []discord.EmbedField{
						{Name: "User Count:", Value: strconv.FormatInt(users, 10), Inline: impls.InlineTrue()},
						{Name: "Team Count:", Value: strconv.FormatInt(teams, 10), Inline: impls.InlineTrue()},
						{Name: "Bot Count:", Value: strconv.FormatInt(bots, 10), Inline: impls.InlineTrue()},
						{Name: "Server Count:", Value: strconv.FormatInt(guilds, 10), Inline: impls.InlineTrue()},
						{Name: "Pack Count:", Value: strconv.FormatInt(packs, 10), Inline: impls.InlineTrue()},
						{Name: "Approved Bots:", Value: strconv.FormatInt(byType["approved"], 10), Inline: impls.InlineTrue()},
						{Name: "Denied Bots:", Value: strconv.FormatInt(byType["denied"], 10), Inline: impls.InlineTrue()},
						{Name: "Certified Bots:", Value: strconv.FormatInt(byType["certified"], 10), Inline: impls.InlineTrue()},
						{Name: "Test Bots (hidden):", Value: strconv.FormatInt(byType["testbot"], 10), Inline: impls.InlineTrue()},
					},
				}},
			})
		},
	}
}

// cmdInfo reports build metadata.
//
// DEVIATION (§14c): upstream reported the Rust toolchain version, the vergen git
// sha/semver/commit message, the build CPU brand and the cargo profile. Go has no
// equivalent of most of those. The Go toolchain version and the VCS revision and
// dirty flag come from debug.ReadBuildInfo; the fields that no longer have a
// meaning are reported as "unknown".
func cmdInfo() *Command {
	return &Command{
		Name:        "info",
		Category:    "Stats",
		Description: "Bot information",
		Run: func(c *Ctx) error {
			var (
				revision = "unknown"
				modified = "unknown"
				version  = "unknown"
			)

			if bi, ok := debug.ReadBuildInfo(); ok {
				version = bi.Main.Version

				for _, setting := range bi.Settings {
					switch setting.Key {
					case "vcs.revision":
						revision = setting.Value
					case "vcs.modified":
						modified = setting.Value
					}
				}
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title: "Bot Information:",
					Color: impls.ColourGreen,
					Fields: []discord.EmbedField{
						{Name: "Bot Version:", Value: version, Inline: impls.InlineTrue()},
						{Name: "Go Version:", Value: runtime.Version(), Inline: impls.InlineTrue()},
						{Name: "Git Commit:", Value: revision, Inline: impls.InlineTrue()},
						{Name: "Modified:", Value: modified, Inline: impls.InlineTrue()},
						{Name: "Built On:", Value: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), Inline: impls.InlineTrue()},
					},
				}},
			})
		},
	}
}

func cmdLeaderboard() *Command {
	return &Command{
		Name:        "leaderboard",
		Category:    "Leaderboard",
		Description: "Top reviewers by approvals and denials",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{Name: "limit", Description: "Limit the amount of results."},
		},
		Run: func(c *Ctx) error {
			limit := 5

			if raw := c.Option("limit", 0); raw != "" {
				parsed, err := strconv.Atoi(raw)

				if err == nil {
					limit = parsed
				}
			}

			stats, err := tasks.QueryTopReviewers(c.Context, limit)

			if err != nil {
				return err
			}

			var sb strings.Builder

			sb.WriteString("Oh, hello there! Let's see who's been fighting bots the most :eyes:\n\n")

			for i, stat := range stats {
				fmt.Fprintf(&sb, "%s <@%s> | Approved: %d | Denied: %d | Total: **%d**\n",
					medal(i), stat.UserID, stat.ApprovedCount, stat.DeniedCount, stat.TotalCount)
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Staff Leaderboard",
					Description: sb.String(),
					Color:       impls.ColourGreen,
				}},
			})
		},
	}
}

// medal decorates the top three positions.
func medal(index int) string {
	switch index {
	case 0:
		return "🥇"
	case 1:
		return "🥈"
	case 2:
		return "🥉"
	default:
		return fmt.Sprintf("%d.", index+1)
	}
}

func cmdRefresh() *Command {
	return &Command{
		Name:        "refresh",
		Category:    "Leaderboard",
		Description: "Force refresh the top reviewer roles",
		Checks:      []Check{staffServer},
		Run: func(c *Ctx) error {
			if err := requirePerm(c, "arcadia.force_refresh_top"); err != nil {
				return err
			}

			// The command uses a limit of 3; the weekly task uses 0. See
			// CONFORMANCE.md.
			stats, err := tasks.QueryTopReviewers(c.Context, 3)

			if err != nil {
				return err
			}

			if _, ok := state.Discord.Caches().Guild(state.Config.Servers.Main); !ok {
				return fmt.Errorf("Failed to get guild")
			}

			if err := tasks.SyncTopReviewerRoles(c.Context, stats); err != nil {
				return err
			}

			return c.Say("**Force Refresh**\nSynced Top Reviewers!")
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

				return c.Say("You are the owner/additional owner of a certified bot! Giving you certified role")
			}

			err = impls.RemoveRole(state.Config.Servers.Main, c.Author.ID, state.Config.Roles.CertifiedDeveloper, "Autorole due to bots owned")

			if err != nil {
				return err
			}

			return c.Say("You are the owner/additional owner of an approved bot! Giving you approved role")
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

// requirePerm resolves the caller's permissions and checks one of them.
func requirePerm(c *Ctx, perm string) error {
	sp, err := impls.GetUserPerms(c.Context, c.Author.ID.String())

	if err != nil {
		return err
	}

	if !perms.HasPerm(sp.Resolve(), perms.PermissionFromString(perm)) {
		return fmt.Errorf("You do not have permission to use this command")
	}

	return nil
}

// runRPC is the shared path from a Discord command into the RPC layer.
func runRPC(c *Ctx, method types.RPCMethod) (rpc.Success, error) {
	return rpc.Execute(c.Context, method, rpc.Handle{
		UserID:     c.Author.ID.String(),
		TargetType: types.TargetTypeBot,
	})
}

// runRPCWithTarget executes an RPC against an explicit target type. The modal
// driver is the only caller that needs a target type other than Bot.
func runRPCWithTarget(c *Ctx, method types.RPCMethod, targetType types.TargetType) (rpc.Success, error) {
	return rpc.Execute(c.Context, method, rpc.Handle{
		UserID:     c.Author.ID.String(),
		TargetType: targetType,
	})
}
