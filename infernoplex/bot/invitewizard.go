package bot

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"popplio/infernoplex/dclient"
	"popplio/infernoplex/invite"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

type inviteSession struct {
	AuthorID   string
	GuildID    snowflake.ID
	ExpiresAt  time.Time
	OnResolved func(ctx context.Context, c *Ctx, invite string) error
}

var inviteSessions = map[string]*inviteSession{}

func startInviteWizard(c *Ctx, onResolved func(ctx context.Context, c *Ctx, invite string) error) error {
	sessionsMu.Lock()
	inviteSessions[c.Author.ID.String()] = &inviteSession{
		AuthorID:   c.Author.ID.String(),
		GuildID:    c.GuildID,
		ExpiresAt:  time.Now().Add(wizardTTL),
		OnResolved: onResolved,
	}
	sessionsMu.Unlock()

	return c.Send(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "Invite Setup",
			Description: "Okay! Now, let's setup the invite for this server! To get started, choose which type of invite you would like\n\n" +
				"- **Invite URL** - Use a (permanent) invite link of your choice\n" +
				"- **Per-User Invite** - Omniplex will create an invite for this server for each user\n" +
				"- **None** - This server will not be invitable. Useful, if you wish to use a whitelist form and manually send out invites",
		}},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewPrimaryButton("Invite URL", "invite:invite_url"),
				discord.NewPrimaryButton("Per-User Invite", "invite:per_user"),
				discord.NewPrimaryButton("No Invites", "invite:none"),
			),
			discord.NewActionRow(
				discord.NewDangerButton("Cancel", "invite:cancel"),
			),
		},
	})
}

func peekInviteSession(userID string) *inviteSession {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	s, ok := inviteSessions[userID]

	if !ok {
		return nil
	}

	if time.Now().After(s.ExpiresAt) {
		delete(inviteSessions, userID)
		return nil
	}

	return s
}

func takeInviteSession(userID string) *inviteSession {
	s := peekInviteSession(userID)

	if s != nil {
		sessionsMu.Lock()
		delete(inviteSessions, userID)
		sessionsMu.Unlock()
	}

	return s
}

func handleInviteComponent(ctx context.Context, e *events.ComponentInteractionCreate, id string) {
	userID := e.User().ID.String()

	s := peekInviteSession(userID)

	if s == nil || s.AuthorID != userID {
		return
	}

	logFailedEdit("clear invite wizard buttons", clearComponents(e.Channel().ID(), e.Message.ID))

	switch id {
	case "invite:cancel":
		takeInviteSession(userID)
		logFailedEdit("answer invite cancel", e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{{Description: "Cancelled."}},
		}))

	case "invite:none":
		takeInviteSession(userID)
		c := ctxFromComponent(ctx, e)

		if err := s.OnResolved(ctx, c, "none"); err != nil {
			c.Fail(err.Error())
		}

	case "invite:invite_url":
		modal := discord.ModalCreate{
			CustomID: "invite:url",
			Title:    "Invite URL Selection",
			Components: []discord.ContainerComponent{
				discord.NewActionRow(
					discord.NewTextInput("invite_url", discord.TextInputStyleShort, "Enter Invite URL").
						WithPlaceholder("Please enter the Invite URL you wish to use!").
						WithMinLength(20).WithMaxLength(100).WithRequired(true),
				),
			},
		}

		logFailedEdit("open invite URL modal", e.Modal(modal))

	case "invite:per_user":
		modal := discord.ModalCreate{
			CustomID: "invite:peruser",
			Title:    "Per-User Invite Selection",
			Components: []discord.ContainerComponent{
				discord.NewActionRow(
					discord.NewTextInput("channel_id", discord.TextInputStyleShort, "Enter Channel ID").
						WithPlaceholder("Please enter the Channel ID you wish to use!").
						WithMinLength(18).WithMaxLength(18).WithRequired(true),
				),
				discord.NewActionRow(
					discord.NewTextInput("max_uses", discord.TextInputStyleShort, "Max Uses").
						WithPlaceholder("How many times should a per-user invite be usable for. Use 1 if unsure").
						WithMinLength(1).WithMaxLength(3).WithRequired(true),
				),
				discord.NewActionRow(
					discord.NewTextInput("max_age", discord.TextInputStyleShort, "Max Age").
						WithPlaceholder("How long should the invite be valid for. Use 0 if unsure").
						WithMinLength(1).WithMaxLength(3).WithRequired(true),
				),
			},
		}

		logFailedEdit("open per-user invite modal", e.Modal(modal))
	}
}

func handleInviteURLModal(ctx context.Context, e *events.ModalSubmitInteractionCreate) {
	userID := e.User().ID.String()

	s := takeInviteSession(userID)

	if s == nil || s.AuthorID != userID {
		return
	}

	c := ctxFromModal(ctx, e)

	if err := c.Defer(); err != nil {
		state.Logger.Error("Failed to defer invite URL modal", zap.Error(err))
		return
	}

	inviteURL := e.Data.Text("invite_url")

	if err := invite.ResolveInvite(ctx, s.GuildID, inviteURL); err != nil {
		c.Fail(fmt.Sprintf("This invite could not be resolved: %s", err))
		return
	}

	if err := s.OnResolved(ctx, c, "invite_url:"+inviteURL); err != nil {
		c.Fail(err.Error())
	}
}

func handleInvitePerUserModal(ctx context.Context, e *events.ModalSubmitInteractionCreate) {
	userID := e.User().ID.String()

	s := takeInviteSession(userID)

	if s == nil || s.AuthorID != userID {
		return
	}

	c := ctxFromModal(ctx, e)

	if err := c.Defer(); err != nil {
		state.Logger.Error("Failed to defer per-user invite modal", zap.Error(err))
		return
	}

	channelID, err := snowflake.Parse(e.Data.Text("channel_id"))

	if err != nil {
		c.Fail("Invalid Channel ID")
		return
	}

	channel, err := dclient.Get().Rest().GetChannel(channelID)

	if err != nil {
		c.Fail(fmt.Sprintf("Failed to fetch channel: %s", err))
		return
	}

	guildChannel, ok := channel.(discord.GuildChannel)

	if !ok {
		c.Fail("Channel must be a guild channel")
		return
	}

	if guildChannel.GuildID() != s.GuildID {
		c.Fail("Channel must be in this server")
		return
	}

	maxUses, err := strconv.Atoi(e.Data.Text("max_uses"))

	if err != nil {
		c.Fail("Invalid Max Uses")
		return
	}

	maxAge, err := strconv.Atoi(e.Data.Text("max_age"))

	if err != nil {
		c.Fail("Invalid Max Age")
		return
	}

	inviteStr := fmt.Sprintf("per_user:%s:%d:%d", channelID, maxUses, maxAge)

	if err := s.OnResolved(ctx, c, inviteStr); err != nil {
		c.Fail(err.Error())
	}
}
