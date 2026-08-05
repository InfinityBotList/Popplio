package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// The verdicts on a claimed bot, gated by review_bots: approved, denied, or
// pulled back into the queue for another look.
//
// Approve is the heaviest handler in the package and the only one that does
// anything after the database write — roles for the owners, a kick from the
// testing server, and the invite URL it hands back.

func approve(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType     string
		claimedBy   *string
		lastClaimed *time.Time
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, last_claimed FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &lastClaimed)

	if err != nil {
		return Success{}, err
	}

	if botType != "pending" {
		return Success{}, errors.New("Entity is not pending review?")
	}

	if claimedBy == nil || *claimedBy == "" || lastClaimed == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed? Do ``/claim`` to claim this bot first!", m.TargetID)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return Success{}, err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE bots SET type = 'approved', claimed_by = NULL WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	// The mod-log post stays INSIDE the transaction, as upstream had it: a failed
	// post still rolls the approval back.
	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Approved!",
			URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend.Parse(), m.TargetID),
			Description: fmt.Sprintf("<@!%s> has approved <@!%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Feedback", Value: m.Reason, Inline: impls.InlineTrue()},
				{Name: "Moderator", Value: "<@!" + h.UserID + ">", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Well done, young traveller!"),
			Color:  impls.ColourGreen,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Success{}, err
	}

	managers, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	for _, owner := range managers.All() {
		ownerSnow, err := snowflake.Parse(owner)

		if err != nil {
			return Success{}, err
		}

		if impls.MemberOnGuild(state.Config.Servers.Main, ownerSnow) {
			if err := impls.AddRole(state.Config.Servers.Main, ownerSnow, state.Config.Roles.BotDeveloper, "Autorole due to bots owned"); err != nil {
				state.Logger.Error("Failed to add role to user", zap.Error(err), zap.String("userID", owner))
			}
		}
	}

	targetSnow, err := snowflake.Parse(m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if impls.MemberOnGuild(state.Config.Servers.Testing, targetSnow) {
		if err := impls.KickMember(state.Config.Servers.Testing, targetSnow, "Bot approved"); err != nil {
			state.Logger.Error("Failed to kick bot from testing server", zap.Error(err), zap.String("botID", m.TargetID))
		}
	}

	var clientID string

	if err := state.Pool.QueryRow(ctx, "SELECT client_id FROM bots WHERE bot_id = $1", m.TargetID).Scan(&clientID); err != nil {
		return Success{}, err
	}

	return Content(fmt.Sprintf(
		"**Invite URL:** https://discord.com/api/v10/oauth2/authorize?client_id=%s&permissions=0&scope=bot%%20applications.commands",
		clientID,
	)), nil
}

func deny(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType     string
		claimedBy   *string
		owner       *string
		lastClaimed *time.Time
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, owner, last_claimed FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &owner, &lastClaimed)

	if err != nil {
		return Success{}, err
	}

	// Leading space is upstream's: the entity name was dropped in a refactor.
	if botType != "pending" {
		return Success{}, errors.New(" is not pending review?")
	}

	if claimedBy == nil || *claimedBy == "" || lastClaimed == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed? Do ``/claim`` to claim this bot first!", m.TargetID)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'denied', claimed_by = NULL WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Denied!",
			URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend.Parse(), m.TargetID),
			Description: fmt.Sprintf("<@%s> has denied <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@!" + h.UserID + ">", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Well done, young traveller at getting denied from the club!"),
			Color:  impls.ColourGreen,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func unverify(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	var botType string

	if err := state.Pool.QueryRow(ctx, "SELECT type FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType); err != nil {
		return Success{}, err
	}

	if botType == "certified" {
		return Success{}, errors.New("Certified bots cannot be unverified")
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'pending', claimed_by = NULL WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	// QUIRK (reproduced): the third field has an EMPTY name, which the Discord API
	// rejects, so this embed post fails and the whole call errors. See
	// CONFORMANCE.md.
	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__ Unverified For Futher Review!__",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "", Value: "<@!" + m.TargetID + ">", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Gonna be pending further review..."),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
