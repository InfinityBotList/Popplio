package bot

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"popplio/arcadia/impls"
	"popplio/arcadia/tasks"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// The read-only reporting commands: site analytics, build metadata and the
// reviewer leaderboard. None of them write anything.

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
					Title:       "Omniplex Analytics",
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
