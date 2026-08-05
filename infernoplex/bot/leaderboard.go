package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

func cmdLeaderboard() *Command {
	return &Command{
		Name:        "leaderboard",
		Description: "Get the users who have voted the most for your server",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{Name: "limit", Description: "How many results to render."},
			discord.ApplicationCommandOptionBool{Name: "filter_onlycurrmembers", Description: "Filter to only those in the server?"},
		},
		Run: func(c *Ctx) error {
			if c.GuildID == 0 {
				return errors.New("This command can only be used in a server.")
			}

			limit := int64(10)

			if raw := c.Option("limit"); raw != "" {
				if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
					limit = n
				}
			}

			if c.Option("filter_onlycurrmembers") == "true" {
				return errors.New("This command requires the GUILD_MEMBERS intent.")
			}

			rows, err := state.Pool.Query(c.Context, `
				SELECT author,
				(SUM(CASE WHEN upvote THEN 1 ELSE 0 END) - SUM(CASE WHEN NOT upvote THEN 1 ELSE 0 END)) AS score
				FROM entity_votes
				WHERE target_id = $1
				AND target_type = 'server'
				AND void = false
				GROUP BY author
				ORDER BY score DESC
			`, c.GuildID.String())

			if err != nil {
				return err
			}

			type row struct {
				author string
				score  int64
			}

			var leaderboard []row

			for rows.Next() {
				var r row

				if err := rows.Scan(&r.author, &r.score); err != nil {
					rows.Close()
					return err
				}

				leaderboard = append(leaderboard, r)

				if int64(len(leaderboard)) >= limit {
					break
				}
			}

			rows.Close()

			if err := rows.Err(); err != nil {
				return err
			}

			if len(leaderboard) == 0 {
				return c.Send(discord.MessageCreate{
					Embeds: []discord.Embed{{
						Title:       "No Votes Yet",
						Description: "Unfortuently, your server has no votes at this time.",
					}},
				})
			}

			var sb strings.Builder

			page := 1

			for i, r := range leaderboard {
				next := fmt.Sprintf("%d. <@%s> - %d\n", i+1, r.author, r.score)

				if sb.Len()+len(next) > 4000 {
					if err := c.Send(discord.MessageCreate{
						Embeds: []discord.Embed{{
							Title:       fmt.Sprintf("Leaderboard (Page %d)", page),
							Description: sb.String(),
						}},
					}); err != nil {
						return err
					}

					page++
					sb.Reset()
				}

				sb.WriteString(next)
			}

			if sb.Len() > 0 {
				return c.Send(discord.MessageCreate{
					Embeds: []discord.Embed{{
						Title:       fmt.Sprintf("Leaderboard (Page %d)", page+1),
						Description: sb.String(),
					}},
				})
			}

			return nil
		},
	}
}
