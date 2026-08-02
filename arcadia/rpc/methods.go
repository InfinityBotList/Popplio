package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// handleMethod is the low-level dispatcher. Every embed below - titles (leading
// spaces included), descriptions, field names, inline flags, footers and colours
// - is reproduced verbatim from the Rust source. The mod-log channel is read by
// humans and several titles have odd leading spaces that are intentional by
// accident.
func handleMethod(ctx context.Context, method types.RPCMethod, h Handle) (Success, error) {
	switch {
	case method.Claim != nil:
		return claim(ctx, method.Claim, h)
	case method.Unclaim != nil:
		return unclaim(ctx, method.Unclaim, h)
	case method.Approve != nil:
		return approve(ctx, method.Approve, h)
	case method.Deny != nil:
		return deny(ctx, method.Deny, h)
	case method.Unverify != nil:
		return unverify(ctx, method.Unverify, h)
	case method.PremiumAdd != nil:
		return premiumAdd(ctx, method.PremiumAdd, h)
	case method.PremiumRemove != nil:
		return premiumRemove(ctx, method.PremiumRemove, h)
	case method.VoteBanAdd != nil:
		return voteBanSet(ctx, method.VoteBanAdd, h, true)
	case method.VoteBanRemove != nil:
		return voteBanSet(ctx, method.VoteBanRemove, h, false)
	case method.VoteReset != nil:
		return voteReset(ctx, method.VoteReset, h)
	case method.VoteResetAll != nil:
		return voteResetAll(ctx, method.VoteResetAll, h)
	case method.ForceRemove != nil:
		return forceRemove(ctx, method.ForceRemove, h)
	case method.CertifyAdd != nil:
		return certifyAdd(ctx, method.CertifyAdd, h)
	case method.CertifyRemove != nil:
		return certifyRemove(ctx, method.CertifyRemove, h)
	case method.BotTransferOwnershipUser != nil:
		return transferOwnershipUser(ctx, method.BotTransferOwnershipUser, h)
	case method.BotTransferOwnershipTeam != nil:
		return transferOwnershipTeam(ctx, method.BotTransferOwnershipTeam, h)
	case method.AppBanUser != nil:
		return appBanSet(ctx, method.AppBanUser, h, true)
	case method.AppUnbanUser != nil:
		return appBanSet(ctx, method.AppUnbanUser, h, false)
	default:
		return Success{}, errors.New("This method does not support this target type yet")
	}
}

func claim(ctx context.Context, m *types.RPCClaim, h Handle) (Success, error) {
	var (
		botType   string
		claimedBy *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy)

	if err != nil {
		return Success{}, err
	}

	if botType != "pending" {
		return Success{}, errors.New("This bot is not pending review")
	}

	// DEAD BRANCH (reproduced): unreachable because the check above already
	// rejected everything that is not "pending". See CONFORMANCE.md.
	if botType == "testbot" {
		return Success{}, errors.New("This bot is a test bot")
	}

	if !m.Force && claimedBy != nil {
		return Success{}, fmt.Errorf("This bot is already claimed by <@%s>", *claimedBy)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET last_claimed = NOW(), claimed_by = $1 WHERE bot_id = $2", h.UserID, m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "claimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Claimed!",
			Description: fmt.Sprintf("<@%s> has claimed <@%s>", h.UserID, m.TargetID),
			Color:       impls.ColourBlurple,
			Fields: []discord.EmbedField{
				{Name: "Force Claim", Value: strconv.FormatBool(m.Force), Inline: impls.InlineFalse()},
			},
			Footer: impls.Footer("This is completely normal, don't worry!"),
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func unclaim(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType   string
		claimedBy *string
		owner     *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &owner)

	if err != nil {
		return Success{}, err
	}

	if botType == "testbot" {
		return Success{}, errors.New("This bot is a test bot")
	}

	if botType != "pending" {
		return Success{}, errors.New("This bot is not pending review")
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if claimedBy == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed", m.TargetID)
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "unclaimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	// No colour is set on this embed upstream.
	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Unclaimed!",
			Description: fmt.Sprintf("<@%s> has unclaimed <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineFalse()},
			},
			Footer: impls.Footer("This is completely normal, don't worry!"),
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

// borealisCacheServer is the Borealis response shape.
type borealisCacheServer struct {
	GuildID    string `json:"guild_id"`
	Name       string `json:"name"`
	InviteCode string `json:"invite_code"`
	Added      bool   `json:"added"`
}

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

	// ORDERING HAZARD (reproduced): the Borealis call and the Discord message both
	// happen INSIDE the transaction, before COMMIT. See CONFORMANCE.md.
	csr, err := addBotToCacheServer(ctx, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Approved!",
			URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend.Parse(), m.TargetID),
			Description: fmt.Sprintf("<@!%s> has approved <@!%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Cache Server", Value: fmt.Sprintf("[%s](https://discord.gg/%s)", csr.Name, csr.InviteCode), Inline: impls.InlineTrue()},
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
		"**Cache Server Invite:** %s\n**Invite URL:** https://discord.com/api/v10/oauth2/authorize?client_id=%s&permissions=0&scope=bot%%20applications.commands&guild_id=%s",
		"https://discord.gg/"+csr.InviteCode,
		clientID,
		csr.GuildID,
	)), nil
}

func addBotToCacheServer(ctx context.Context, botID string) (borealisCacheServer, error) {
	url := fmt.Sprintf("%s/addBotToCacheServer?bot_id=%s&ignore_bot_type=true", state.Config.Arcadia.BorealisURL, botID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)

	if err != nil {
		return borealisCacheServer{}, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return borealisCacheServer{}, err
	}

	defer resp.Body.Close()

	var csr borealisCacheServer

	if err := json.NewDecoder(resp.Body).Decode(&csr); err != nil {
		return borealisCacheServer{}, fmt.Errorf("Error decoding borealis response: %v", err)
	}

	return csr, nil
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
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
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
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
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
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
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

func premiumAdd(ctx context.Context, m *types.RPCPremiumAdd, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	_, err := state.Pool.Exec(ctx,
		"UPDATE bots SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE bot_id = $2",
		m.TimePeriodHours, m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Premium Added!",
			Description: fmt.Sprintf("<@%s> has added premium to <@%s> for %d hours", h.UserID, m.TargetID, m.TimePeriodHours),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Well done, young traveller! Use it wisely..."),
			Color:  impls.ColourGreen,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func premiumRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET premium = false WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Premium Removed!",
			Description: fmt.Sprintf("<@%s> has removed premium from <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Well done, young traveller. Sad to see you go..."),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func voteBanSet(ctx context.Context, m *types.RPCTargetReason, h Handle, banned bool) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	query := "UPDATE bots SET vote_banned = false WHERE bot_id = $1"
	if banned {
		query = "UPDATE bots SET vote_banned = true WHERE bot_id = $1"
	}

	if _, err := state.Pool.Exec(ctx, query, m.TargetID); err != nil {
		return Success{}, err
	}

	title := "Vote Ban Removed!"
	description := fmt.Sprintf("<@%s> has removed the vote ban on <@%s>", h.UserID, m.TargetID)

	if banned {
		title = "Vote Ban Edit!"
		description = fmt.Sprintf("<@%s> has set the vote ban on <@%s>", h.UserID, m.TargetID)
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       title,
			Description: description,
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Remember: don't abuse our services!"),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func voteReset(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	_, err := state.Pool.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (single entity) reset', voided_at = NOW() WHERE target_type = $1 AND target_id = $2 AND void = FALSE",
		h.TargetType.String(), m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__Entity Vote Reset!__",
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Target ID", Value: m.TargetID, Inline: impls.InlineTrue()},
				{Name: "Target Type", Value: h.TargetType.String(), Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Sad life :("),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func voteResetAll(ctx context.Context, m *types.RPCVoteResetAll, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return Success{}, err
	}

	defer tx.Rollback(ctx)

	// Note: unlike VoteReset this has no void = FALSE filter.
	_, err = tx.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (all entities) reset', voided_at = NOW() WHERE target_type = $1 AND immutable = false",
		h.TargetType.String(),
	)

	if err != nil {
		return Success{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__All Entity Votes Reset!__",
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Target Type", Value: h.TargetType.String(), Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Sad life :("),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func forceRemove(ctx context.Context, m *types.RPCForceRemove, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	targetSnow, err := snowflake.Parse(m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if m.Kick && isProtectedBot(targetSnow) {
		return Success{}, errors.New("You can't force delete this bot with 'kick' enabled!")
	}

	if _, err := state.Pool.Exec(ctx, "DELETE FROM bots WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       " Force Deleted!",
			Description: fmt.Sprintf("<@%s> has force-removed <@%s> for violating our rules or Discord ToS", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Remember: don't abuse our services!"),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	if m.Kick && impls.MemberOnGuild(state.Config.Servers.Main, targetSnow) {
		if err := impls.KickMember(state.Config.Servers.Main, targetSnow, "Force deleted via RPC with kick set to true"); err != nil {
			return Success{}, err
		}
	}

	return NoContent(), nil
}

func isProtectedBot(id snowflake.ID) bool {
	for _, protected := range state.Config.Arcadia.ProtectedBots {
		if protected == id {
			return true
		}
	}

	return false
}

func certifyAdd(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'certified' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       " Force Certified!",
			Description: fmt.Sprintf("<@%s> has force-certified <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Neat"),
			Color:  impls.ColourRedLower,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func certifyRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'approved' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       " Uncertified!",
			Description: fmt.Sprintf("<@%s> has uncertified <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Uh oh, looks like you've been naughty..."),
			Color:  impls.ColourRedLower,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func transferOwnershipUser(ctx context.Context, m *types.RPCBotTransferOwnershipUser, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	var teamOwner *uuid.UUID

	if err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&teamOwner); err != nil {
		return Success{}, err
	}

	if teamOwner != nil {
		return Success{}, errors.New(" is in a team. Please use BotTransferOwnershipTeam")
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET owner = $2 WHERE bot_id = $1", m.TargetID, m.NewOwner); err != nil {
		return Success{}, err
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       " Ownership Force Update!",
			Description: fmt.Sprintf("<@%s> has force-updated the ownership of <@%s> to <@%s>", h.UserID, m.TargetID, m.NewOwner),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Contact support if you think this is a mistake"),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func transferOwnershipTeam(ctx context.Context, m *types.RPCBotTransferOwnershipTeam, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := botExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	teamID, err := uuid.Parse(m.NewTeam)

	if err != nil {
		return Success{}, errors.New("Invalid team ID")
	}

	var teamOwner *uuid.UUID

	if err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&teamOwner); err != nil {
		return Success{}, err
	}

	if teamOwner == nil {
		return Success{}, errors.New(" is not in a team. Please use TransferOwnership")
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET team_owner = $2 WHERE bot_id = $1", m.TargetID, teamID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       " Ownership Force Update!",
			Description: fmt.Sprintf("<@%s> has force-updated the ownership of <@%s> to team %s", h.UserID, m.TargetID, teamID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Contact support if you think this is a mistake"),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func appBanSet(ctx context.Context, m *types.RPCTargetReason, h Handle, banned bool) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	if err := userExists(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	query := "UPDATE users SET app_banned = false WHERE user_id = $1"
	if banned {
		query = "UPDATE users SET app_banned = true WHERE user_id = $1"
	}

	if _, err := state.Pool.Exec(ctx, query, m.TargetID); err != nil {
		return Success{}, err
	}

	title := "[Apps] Unbanned User"
	description := fmt.Sprintf("<@%s> has unbanned <@%s> from using apps.", h.UserID, m.TargetID)
	footer := "Welcome, back!"

	if banned {
		title = "[Apps] Banned User"
		description = fmt.Sprintf("<@%s> has banned <@%s> from using apps.", h.UserID, m.TargetID)
		footer = "Well done, young traveller. Sad to see you go..."
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       title,
			Description: description,
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer(footer),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

// staffGeneralLog writes the claim/unclaim audit row.
func staffGeneralLog(ctx context.Context, userID, action, targetID string, claimedByPrev *string) error {
	data, err := json.Marshal(map[string]any{
		"target_id":       targetID,
		"claimed_by_prev": claimedByPrev,
	})

	if err != nil {
		return err
	}

	_, err = state.Pool.Exec(ctx, "INSERT INTO staff_general_logs (user_id, action, data) VALUES ($1, $2, $3)", userID, action, data)

	return err
}
