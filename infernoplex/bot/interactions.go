package bot

import (
	"context"
	"strings"
	"sync"
	"time"

	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

const wizardTTL = 6 * time.Minute

var sessionsMu sync.Mutex

func onComponent(ctx context.Context, e *events.ComponentInteractionCreate) {
	id := e.Data.CustomID()

	switch {
	case strings.HasPrefix(id, "setup:"):
		handleSetupComponent(ctx, e, id)
	case strings.HasPrefix(id, "invite:"):
		handleInviteComponent(ctx, e, id)
	case strings.HasPrefix(id, "update:"):
		handleUpdateComponent(ctx, e, id)
	case strings.HasPrefix(id, "delete:"):
		handleDeleteComponent(ctx, e, id)
	}
}

func onModalSubmit(ctx context.Context, e *events.ModalSubmitInteractionCreate) {
	switch e.Data.CustomID {
	case "setup:info":
		handleSetupInfoModal(ctx, e)
	case "invite:url":
		handleInviteURLModal(ctx, e)
	case "invite:peruser":
		handleInvitePerUserModal(ctx, e)
	case "update:basic":
		handleUpdateBasicModal(ctx, e)
	}
}

func ctxFromComponent(ctx context.Context, e *events.ComponentInteractionCreate) *Ctx {
	c := &Ctx{
		Context:   ctx,
		Author:    e.User(),
		ChannelID: e.Channel().ID(),
		Options:   map[string]string{},
		component: e,
	}

	if e.GuildID() != nil {
		c.GuildID = *e.GuildID()
	}

	return c
}

func ctxFromModal(ctx context.Context, e *events.ModalSubmitInteractionCreate) *Ctx {
	c := &Ctx{
		Context:   ctx,
		Author:    e.User(),
		ChannelID: e.Channel().ID(),
		Options:   map[string]string{},
		modal:     e,
	}

	if e.GuildID() != nil {
		c.GuildID = *e.GuildID()
	}

	return c
}

func clearComponents(channelID, messageID snowflake.ID) error {
	empty := []discord.ContainerComponent{}
	_, err := dclient.Get().Rest().UpdateMessage(channelID, messageID, discord.MessageUpdate{Components: &empty})
	return err
}

func logFailedEdit(what string, err error) {
	if err != nil {
		state.Logger.Error("Failed to "+what, zap.Error(err))
	}
}
