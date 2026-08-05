package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// onComponent drives the queue pager and the claim prompt.
func onComponent(ctx context.Context, e *events.ComponentInteractionCreate) {
	id := e.Data.CustomID()
	messageID := e.Message.ID.String()

	state.Logger.Info(fmt.Sprintf("Received interaction: %s", id))

	c := &Ctx{
		Context:   ctx,
		Author:    e.User(),
		ChannelID: e.Channel().ID(),
		GuildID:   guildOrZero(e.GuildID()),
		Options:   map[string]string{},
	}

	switch {
	case strings.HasPrefix(id, "q:"):
		handleQueueButton(c, e, id, messageID)
	case id == "fclaim" || id == "remind":
		handleClaimButton(c, e, id, messageID)
	case strings.HasPrefix(id, "perm:"):
		handlePermEditor(c, e, id, messageID)
	}
}

func handleQueueButton(c *Ctx, e *events.ComponentInteractionCreate, id, messageID string) {
	sessionsMu.Lock()
	session, ok := queueSessions[messageID]
	sessionsMu.Unlock()

	if !ok {
		return
	}

	// Author-scoped, like poise's author_id filter.
	if session.AuthorID != e.User().ID.String() {
		return
	}

	if time.Now().After(session.ExpiresAt) {
		sessionsMu.Lock()
		delete(queueSessions, messageID)
		sessionsMu.Unlock()
		return
	}

	if err := e.DeferUpdateMessage(); err != nil {
		state.Logger.Error("Failed to defer queue interaction", zap.Error(err))
		return
	}

	if id == "q:cancel" {
		sessionsMu.Lock()
		delete(queueSessions, messageID)
		sessionsMu.Unlock()

		if err := dclient.Get().Rest().DeleteMessage(e.Channel().ID(), e.Message.ID); err != nil {
			state.Logger.Error("Failed to delete queue message", zap.Error(err))
		}

		return
	}

	// BUG FIXED (flagged): upstream's prev handler is
	// `if current == 0 { current = 0 } current -= 1`, which underflows on the
	// first page. Both bounds are clamped properly here.
	switch id {
	case "q:prev":
		if session.Current > 0 {
			session.Current--
		}
	case "q:next":
		if session.Current < len(session.Bots)-1 {
			session.Current++
		}
	}

	msg, err := renderQueue(c, session)

	if err != nil {
		state.Logger.Error("Failed to render queue", zap.Error(err))
		return
	}

	_, err = dclient.Get().Rest().UpdateMessage(e.Channel().ID(), e.Message.ID, discord.MessageUpdate{
		Content:    &msg.Content,
		Embeds:     &msg.Embeds,
		Components: &msg.Components,
	})

	if err != nil {
		state.Logger.Error("Failed to update queue message", zap.Error(err))
	}
}

func handleClaimButton(c *Ctx, e *events.ComponentInteractionCreate, id, messageID string) {
	sessionsMu.Lock()
	session, ok := claimSessions[messageID]

	if ok {
		delete(claimSessions, messageID)
	}

	sessionsMu.Unlock()

	if !ok || session.AuthorID != e.User().ID.String() {
		return
	}

	if err := e.DeferUpdateMessage(); err != nil {
		state.Logger.Error("Failed to defer claim interaction", zap.Error(err))
		return
	}

	// Remove the buttons after a press.
	empty := []discord.ContainerComponent{}

	if _, err := dclient.Get().Rest().UpdateMessage(e.Channel().ID(), e.Message.ID, discord.MessageUpdate{Components: &empty}); err != nil {
		state.Logger.Error("Failed to clear claim buttons", zap.Error(err))
	}

	if id == "remind" {
		if err := claimReminderLog(c, session.BotID, session.ClaimedBy); err != nil {
			c.Fail(fmt.Sprintf("There was an error running this command: %s", err))
			return
		}

		c.Say(fmt.Sprintf(
			"<@%s>, did you forgot to finish testing <@%s>? This reminder has been recorded internally for staff activity tracking purposes!",
			session.ClaimedBy, session.BotID))

		return
	}

	_, err := runRPC(c, types.RPCMethod{Claim: &types.RPCClaim{TargetID: session.BotID, Force: true}})

	if err != nil {
		c.Fail(fmt.Sprintf("There was an error running this command: %s", err))
		return
	}

	c.Ok("Claimed bot successfully, the bot owner has been informed")
}

// cmdRPC is the modal driver: pick a target type and a method, then fill in a
// modal built from the method's fields.
func cmdRPC() *Command {
	return &Command{
		Name:        "rpc",
		Category:    "RPC",
		Description: "Perform an RPC action",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "target_type",
				Description: "The target type to perform the action on",
				Required:    true,
				Choices:     targetTypeChoices(),
			},
			discord.ApplicationCommandOptionString{
				Name:        "method",
				Description: "The RPC method to run",
				Required:    true,
			},
		},
		Run: func(c *Ctx) error {
			targetType, err := types.TargetTypeFromString(c.Option("target_type", 0))

			if err != nil {
				return fmt.Errorf("Invalid target type")
			}

			methodName := c.Option("method", 1)

			variant, err := types.EmptyRPCMethod(methodName)

			if err != nil {
				return err
			}

			if c.slash == nil {
				return fmt.Errorf("This command must be used as a slash command")
			}

			sessionsMu.Lock()
			rpcSessions[c.Author.ID.String()] = &rpcSession{
				AuthorID:   c.Author.ID.String(),
				TargetType: targetType,
				Method:     methodName,
			}
			sessionsMu.Unlock()

			components := make([]discord.ContainerComponent, 0, len(variant.Fields()))

			for _, field := range variant.Fields() {
				style := discord.TextInputStyleShort

				if field.FieldType == types.FieldTypeTextarea {
					style = discord.TextInputStyleParagraph
				}

				components = append(components, discord.ActionRowComponent{
					discord.NewTextInput(field.ID, style, field.Label).WithPlaceholder(field.Placeholder),
				})
			}

			c.replied = true

			return c.slash.Modal(discord.ModalCreate{
				CustomID:   "rpc:" + methodName,
				Title:      truncate(variant.Label(), 45),
				Components: components,
			})
		},
	}
}

// rpcMethodChoices exposes the 18 RPC methods as slash-command choices. Upstream
// autocompleted them; a choice list is a better fit here because the whole set
// fits inside Discord's 25-choice limit and needs no round trip.
func rpcMethodChoices() []discord.ApplicationCommandOptionChoiceString {
	choices := make([]discord.ApplicationCommandOptionChoiceString, 0, len(types.RPCMethodVariants))

	for _, name := range types.RPCMethodVariants {
		method, err := types.EmptyRPCMethod(name)

		if err != nil {
			continue
		}

		// The label is what staff read; the value is the variant name the
		// dispatcher needs.
		choices = append(choices, discord.ApplicationCommandOptionChoiceString{
			Name:  truncate(method.Label(), 100),
			Value: name,
		})
	}

	return choices
}

func targetTypeChoices() []discord.ApplicationCommandOptionChoiceString {
	choices := make([]discord.ApplicationCommandOptionChoiceString, 0, len(types.TargetTypeVariants))

	for _, t := range types.TargetTypeVariants {
		choices = append(choices, discord.ApplicationCommandOptionChoiceString{
			Name:  t.Variant(),
			Value: t.Variant(),
		})
	}

	return choices
}

// onModalSubmit builds the RPCMethod from the modal fields and executes it.
func onModalSubmit(ctx context.Context, e *events.ModalSubmitInteractionCreate) {
	if !strings.HasPrefix(e.Data.CustomID, "rpc:") {
		return
	}

	userID := e.User().ID.String()

	sessionsMu.Lock()
	session, ok := rpcSessions[userID]

	if ok {
		delete(rpcSessions, userID)
	}

	sessionsMu.Unlock()

	if !ok {
		return
	}

	methodName := strings.TrimPrefix(e.Data.CustomID, "rpc:")

	variant, err := types.EmptyRPCMethod(methodName)

	if err != nil {
		modalReply(e, fmt.Sprintf("There was an error running this command: %s", err), impls.ColourRed)
		return
	}

	fields := make(map[string]any)

	for _, field := range variant.Fields() {
		raw := e.Data.Text(field.ID)

		value, err := coerceField(field, raw)

		if err != nil {
			modalReply(e, fmt.Sprintf("There was an error running this command: %s", err), impls.ColourRed)
			return
		}

		fields[field.ID] = value
	}

	// Rebuild the method from {method_name: fields} through the same codec the
	// panel uses, so the Discord path and the panel path produce identical
	// payloads.
	payload, err := json.Marshal(map[string]any{methodName: fields})

	if err != nil {
		modalReply(e, fmt.Sprintf("There was an error running this command: %s", err), impls.ColourRed)
		return
	}

	var method types.RPCMethod

	if err := json.Unmarshal(payload, &method); err != nil {
		modalReply(e, fmt.Sprintf("There was an error running this command: %s", err), impls.ColourRed)
		return
	}

	c := &Ctx{
		Context:   ctx,
		Author:    e.User(),
		ChannelID: e.Channel().ID(),
		GuildID:   guildOrZero(e.GuildID()),
		Options:   map[string]string{},
	}

	res, err := runRPCWithTarget(c, method, session.TargetType)

	if err != nil {
		modalReply(e, fmt.Sprintf("There was an error running this command: %s", err), impls.ColourRed)
		return
	}

	content, ok := res.Text()

	if !ok {
		content = "Successfully performed this action"
	}

	modalReply(e, content, impls.ColourGreen)
}

// modalReply answers a modal submission. The modal driver cannot use Ctx.Say —
// the interaction has to be answered through the event itself — so this is the
// same embed shape by hand.
func modalReply(e *events.ModalSubmitInteractionCreate, content string, colour int) {
	err := e.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{{
		Description: content,
		Color:       colour,
	}}})

	if err != nil {
		state.Logger.Error("Failed to answer an RPC modal", zap.Error(err))
	}
}

// coerceField converts a modal's raw text into the type the RPC field expects.
func coerceField(field types.RPCField, raw string) (any, error) {
	switch field.FieldType {
	case types.FieldTypeText, types.FieldTypeTextarea:
		return raw, nil
	case types.FieldTypeNumber:
		n, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)

		if err != nil {
			return nil, fmt.Errorf("Invalid number")
		}

		return n, nil
	case types.FieldTypeHour:
		return parseHours(raw)
	case types.FieldTypeBoolean:
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "t", "y":
			return true, nil
		case "false", "f", "n":
			return false, nil
		default:
			return nil, fmt.Errorf("Invalid boolean")
		}
	default:
		return raw, nil
	}
}

// parseHours parses "<n> <unit>" into a number of hours. The space is required.
func parseHours(raw string) (int64, error) {
	number, unit, found := strings.Cut(strings.TrimSpace(raw), " ")

	if !found {
		return 0, fmt.Errorf("Invalid time format. Format must be WITH A SPACE BETWEEN THE NUMBER AND THE UNIT")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(number), 10, 64)

	if err != nil {
		return 0, fmt.Errorf("Invalid time format. Format must be WITH A SPACE BETWEEN THE NUMBER AND THE UNIT")
	}

	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "years", "year", "y":
		return n * 365 * 24, nil
	case "months", "month", "mo", "m":
		return n * 30 * 24, nil
	case "weeks", "week", "w":
		return n * 7 * 24, nil
	case "days", "day", "d":
		return n * 24, nil
	case "hours", "hour", "hrs", "hr", "h":
		return n, nil
	default:
		return 0, fmt.Errorf("Invalid time format. Unit must be years, months, weeks, days or hours")
	}
}

func guildOrZero(id *snowflake.ID) snowflake.ID {
	if id == nil {
		return 0
	}

	return *id
}
