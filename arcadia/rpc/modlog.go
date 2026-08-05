package rpc

import (
	"context"

	"popplio/arcadia/impls"

	"github.com/disgoorg/disgo/discord"
)

// The mod-log embeds, and the guards every handler opens with.
//
// The embed shapes here are frozen (see the package doc), so these helpers build
// exactly what the handlers used to build inline. The titles, descriptions and
// footers deliberately stay at the call sites: that is where they are read, and
// it is where arcadia/conformance looks for them.

// modLogReason posts the embed most actions use — what happened, to what, and
// the reason given. Nine handlers share it.
func modLogReason(title, description, footer string, colour int, reason string) error {
	return impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       title,
			Description: description,
			Fields:      []discord.EmbedField{reasonField(reason)},
			Footer:      impls.Footer(footer),
			Color:       colour,
		}},
	})
}

// reasonField is the "Reason" field the mod log carries on nearly every action.
func reasonField(reason string) discord.EmbedField {
	return discord.EmbedField{Name: "Reason", Value: reason, Inline: impls.InlineTrue()}
}

// guardBot is the opening most handlers share: a reason short enough to store,
// and a bot that exists to apply it to.
//
// The order matters and is the order the handlers had it in — a reason that is
// too long is rejected before the database is touched at all.
func guardBot(ctx context.Context, targetID, reason string) error {
	if err := checkReason(reason); err != nil {
		return err
	}

	return botExists(ctx, targetID)
}

// guardUser is guardBot for the handlers that act on a user rather than an
// entity.
func guardUser(ctx context.Context, targetID, reason string) error {
	if err := checkReason(reason); err != nil {
		return err
	}

	return userExists(ctx, targetID)
}
