package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/infernoplex/dclient"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

type updateBasicSession struct {
	AuthorID  string
	GuildID   snowflake.ID
	ExpiresAt time.Time
}

var updateBasicSessions = map[string]*updateBasicSession{}

func cmdUpdate() *Command {
	return &Command{
		Name:        "update",
		Description: "Update your server information on Omniplex, needs the Edit Servers permission",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "pane",
				Description: "The pane to update",
				Required:    true,
				Choices: []discord.ApplicationCommandOptionChoiceString{
					{Name: "Basic Server Information", Value: "basic"},
					{Name: "Server Invite", Value: "invite"},
				},
			},
		},
		Checks: []Check{
			func(c *Ctx) error {
				return checkForPermission(c.Context, c.GuildID, c.Author.ID.String(), perms.EntityEditServers)
			},
		},
		Run: func(c *Ctx) error {
			if c.GuildID == 0 {
				return errors.New("This command can only be executed in a server")
			}

			switch c.Option("pane") {
			case "basic":
				return startUpdateBasicInfo(c)
			case "invite":
				return startUpdateInvite(c)
			default:
				return errors.New("Invalid pane")
			}
		},
	}
}

func startUpdateBasicInfo(c *Ctx) error {
	sessionsMu.Lock()
	updateBasicSessions[c.Author.ID.String()] = &updateBasicSession{
		AuthorID:  c.Author.ID.String(),
		GuildID:   c.GuildID,
		ExpiresAt: time.Now().Add(wizardTTL),
	}
	sessionsMu.Unlock()

	return c.Send(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Update Server Information",
			Description: "Oh, hello there :eyes:\nI see you want to update your server listing on Omniplex! Let's get started!",
		}},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewPrimaryButton("Next", "update:next"),
				discord.NewDangerButton("Cancel", "update:cancel"),
			),
		},
	})
}

func startUpdateInvite(c *Ctx) error {
	guildID := c.GuildID

	return startInviteWizard(c, func(ctx context.Context, c *Ctx, inviteStr string) error {
		if err := c.Defer(); err != nil {
			return err
		}

		if _, err := state.Pool.Exec(ctx, "UPDATE servers SET invite = $2 WHERE server_id = $1", guildID.String(), inviteStr); err != nil {
			return fmt.Errorf("error while updating invite: %w", err)
		}

		return c.Ok("All done :white_check_mark:")
	})
}

func handleUpdateComponent(ctx context.Context, e *events.ComponentInteractionCreate, id string) {
	userID := e.User().ID.String()

	sessionsMu.Lock()
	s, ok := updateBasicSessions[userID]
	sessionsMu.Unlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		return
	}

	if id == "update:cancel" {
		sessionsMu.Lock()
		delete(updateBasicSessions, userID)
		sessionsMu.Unlock()

		logFailedEdit("clear update buttons", clearComponents(e.Channel().ID(), e.Message.ID))
		logFailedEdit("answer update cancel", e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{{Description: "Cancelled."}},
		}))

		return
	}

	empty := []discord.ContainerComponent{}
	embeds := []discord.Embed{{
		Title:       "Update Server Information",
		Description: "Please follow the instructions given in the modal to continue! If you don't see a modal, please rerun the command",
	}}

	_, editErr := dclient.Get().Rest().UpdateMessage(e.Channel().ID(), e.Message.ID, discord.MessageUpdate{Embeds: &embeds, Components: &empty})
	logFailedEdit("edit update message", editErr)

	modal := discord.ModalCreate{
		CustomID: "update:basic",
		Title:    "Update Server Information",
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewTextInput("short", discord.TextInputStyleShort, "Short Description").
					WithPlaceholder("Something short and snazzy to brag about!").
					WithMinLength(20).WithMaxLength(100).WithRequired(true),
			),
			discord.NewActionRow(
				discord.NewTextInput("long", discord.TextInputStyleParagraph, "Long/Extended Description").
					WithPlaceholder("Both markdown and HTML are supported!").
					WithMinLength(30).WithMaxLength(4000).WithRequired(true),
			),
		},
	}

	logFailedEdit("open update basic info modal", e.Modal(modal))
}

func handleUpdateBasicModal(ctx context.Context, e *events.ModalSubmitInteractionCreate) {
	userID := e.User().ID.String()

	sessionsMu.Lock()
	s, ok := updateBasicSessions[userID]

	if ok {
		delete(updateBasicSessions, userID)
	}

	sessionsMu.Unlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		return
	}

	c := ctxFromModal(ctx, e)

	if err := c.Defer(); err != nil {
		state.Logger.Error("Failed to defer update basic info modal", zap.Error(err))
		return
	}

	short := e.Data.Text("short")
	long := e.Data.Text("long")

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET short = $2, long = $3 WHERE server_id = $1", s.GuildID.String(), short, long); err != nil {
		c.Fail(fmt.Sprintf("Error while updating server: %s", err))
		return
	}

	c.Ok("All done :white_check_mark:")
}
