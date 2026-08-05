package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type deleteSession struct {
	AuthorID  string
	GuildID   snowflake.ID
	ExpiresAt time.Time
}

var deleteSessions = map[string]*deleteSession{}

func cmdDelete() *Command {
	return &Command{
		Name:        "delete",
		Description: "Delete your server from Infinity List, needs the Delete Servers permission",
		Checks: []Check{
			func(c *Ctx) error {
				return checkForPermission(c.Context, c.GuildID, c.Author.ID.String(), perms.EntityDeleteServers)
			},
		},
		Run: func(c *Ctx) error {
			if c.GuildID == 0 {
				return errors.New("This command can only be executed in a server")
			}

			sessionsMu.Lock()
			deleteSessions[c.Author.ID.String()] = &deleteSession{
				AuthorID:  c.Author.ID.String(),
				GuildID:   c.GuildID,
				ExpiresAt: time.Now().Add(wizardTTL),
			}
			sessionsMu.Unlock()

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Confirm Server Deletion?",
					Description: "Are you sure you want to delete your server from Infinity List? This action is irreversible so think before acting!.",
				}},
				Components: []discord.ContainerComponent{
					discord.NewActionRow(
						discord.NewDangerButton("Confirm", "delete:confirm"),
						discord.NewSecondaryButton("Cancel", "delete:cancel"),
					),
				},
			})
		},
	}
}

func handleDeleteComponent(ctx context.Context, e *events.ComponentInteractionCreate, id string) {
	userID := e.User().ID.String()

	sessionsMu.Lock()
	s, ok := deleteSessions[userID]

	if ok {
		delete(deleteSessions, userID)
	}

	sessionsMu.Unlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		return
	}

	logFailedEdit("clear delete confirm buttons", clearComponents(e.Channel().ID(), e.Message.ID))

	if id == "delete:cancel" {
		logFailedEdit("answer delete cancel", e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{{Description: "Cancelled."}},
		}))

		return
	}

	c := ctxFromComponent(ctx, e)

	if err := c.Defer(); err != nil {
		state.Logger.Error("Failed to defer delete confirmation", zap.Error(err))
		return
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		c.Fail(err.Error())
		return
	}

	defer tx.Rollback(ctx)

	var vanityRef pgtype.UUID

	if err := tx.QueryRow(ctx, "SELECT vanity_ref FROM servers WHERE server_id = $1", s.GuildID.String()).Scan(&vanityRef); err != nil {
		c.Fail(fmt.Sprintf("Error while fetching server: %s", err))
		return
	}

	if _, err := tx.Exec(ctx, "DELETE FROM vanity WHERE itag = $1", vanityRef); err != nil {
		c.Fail(fmt.Sprintf("Error while deleting vanity: %s", err))
		return
	}

	if _, err := tx.Exec(ctx, "DELETE FROM servers WHERE server_id = $1", s.GuildID.String()); err != nil {
		c.Fail(fmt.Sprintf("Error while deleting server: %s", err))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.Fail(err.Error())
		return
	}

	c.Ok("All done :white_check_mark:")
}
