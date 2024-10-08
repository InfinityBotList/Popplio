package apps

import (
	"errors"
	"fmt"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"
)

var permBotResubmit = kittycat.Permission{Namespace: "bot", Perm: teams.PermissionResubmit}
var permBotCertify = kittycat.Permission{Namespace: "bot", Perm: teams.PermissionRequestCertification}

var ErrNoPersist = errors.New("no persist") // This error should be returned when the app should not be persisted to the database for review

func extraLogicResubmit(d uapi.RouteData, p types.Position, answers map[string]string) error {
	// Get the bot ID
	botID, ok := answers["id"]

	if !ok {
		return errors.New("bot ID not found")
	}

	// Get the bot
	var botType string
	err := state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

	if err != nil {
		return fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", botID)

	if err != nil {
		return fmt.Errorf("error getting user bot perms: %w", err)
	}

	// Check if user has TeamPermissionResubmitBots
	if !kittycat.HasPerm(perms, permBotResubmit) {
		return errors.New("you do not have permission to resubmit bots")
	}

	if botType == "approved" || botType == "pending" || botType == "certified" {
		return errors.New("bot is approved, pending or certified | state=" + botType)
	}

	// Set the bot type to pending
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET type = 'pending', claimed_by = NULL WHERE bot_id = $1", botID)

	if err != nil {
		return fmt.Errorf("error setting bot type to pending: %w", err)
	}

	user, err := dovewing.GetUser(d.Context, botID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error getting discord user: %w", err)
	}

	// Send an embed to the bot logs channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []discord.Embed{
			{
				Title:       "Bot Resubmitted!",
				URL:         state.Config.Sites.Frontend.Parse() + "/bots/" + botID,
				Description: "User <@" + d.Auth.ID + "> has resubmitted their bot",
				Color:       0x00ff00,
				Fields: []discord.EmbedField{
					{
						Name:  "Bot ID",
						Value: botID,
					},
					{
						Name:  "Bot Name",
						Value: user.DisplayName + " (" + user.ID + ")",
					},
					{
						Name:   "Reason",
						Value:  answers["reason"],
						Inline: validators.TruePtr,
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("error sending embed to bot logs channel: %w", err)
	}

	return nil // Should it be ErrNoPersist?
}

func extraLogicCert(d uapi.RouteData, p types.Position, answers map[string]string) error {
	// Get the bot ID
	botID, ok := answers["id"]

	if !ok {
		return errors.New("bot ID not found")
	}

	// Get the bot
	var botType string
	err := state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

	if err != nil {
		return fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", botID)

	if err != nil {
		return fmt.Errorf("error getting user bot perms: %w", err)
	}

	// Check if user has TeamPermissionCertifyBots
	if !kittycat.HasPerm(perms, permBotCertify) {
		return errors.New("you do not have permission to certify bots")
	}

	if botType != "approved" {
		return errors.New("bot is not approved | state=" + botType)
	}

	// Now check server count and unique clicks
	var serverCount int64
	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT servers, cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", botID).Scan(&serverCount, &uniqueClicks)

	if err != nil {
		return fmt.Errorf("error getting server count: %w", err)
	}

	if serverCount < 100 {
		return errors.New("bot does not have enough servers to be certified: has " + fmt.Sprint(serverCount) + ", needs 100")
	}

	if uniqueClicks < 30 {
		return errors.New("bot does not have enough unique clicks to be certified: has " + fmt.Sprint(uniqueClicks) + ", needs 30")
	}

	return nil
}

func reviewLogicBanAppeal(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	if approve {
		// Unban user

		if len(reason) > 384 {
			return errors.New("reason must be less than 384 characters")
		}

		userId, err := snowflake.Parse(resp.UserID)

		if err != nil {
			return fmt.Errorf("error parsing user ID: %w", err)
		}

		err = state.Discord.Rest().DeleteBan(
			state.Config.Servers.Main,
			userId,
			rest.WithReason("Ban appeal accepted by "+d.Auth.ID+" | "+reason),
		)

		if err != nil {
			return err
		}
	} else {
		// Denial is always possible
		return nil
	}

	return nil
}

func reviewLogicCert(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	if approve {
		// Get the bot ID
		botID, ok := resp.Answers["id"]

		if !ok {
			return errors.New("bot ID not found")
		}

		botIdSnow, err := snowflake.Parse(botID)

		if err != nil {
			return fmt.Errorf("error parsing bot ID: %w", err)
		}

		// Get the bot
		var botType string
		err = state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

		if err != nil {
			return fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
		}

		if botType == "certified" {
			return nil // Just approve the review
		}

		if botType != "approved" {
			return errors.New("bot is not approved | state=" + botType + ". Please deny the certification until approved")
		}

		// Set the bot type to certified
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET type = 'certified' WHERE bot_id = $1", botID)

		if err != nil {
			return fmt.Errorf("error setting bot type to certified: %w", err)
		}

		// Give roles
		err = state.Discord.Rest().AddMemberRole(state.Config.Servers.Main, botIdSnow, state.Config.Roles.CertBot)

		if err != nil {
			return fmt.Errorf("error giving certified bot role to bot, but successfully certified bot: %v", err)
		}

		// Send an embed to the bot logs channel
		_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Bot Certified!",
					URL:         state.Config.Sites.Frontend.Parse() + "/bots/" + botID,
					Description: "<@" + d.Auth.ID + "> has certified bot <@" + botID + ">",
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Bot ID",
							Value: botID,
						},
						{
							Name:  "Reason",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "If you are the owner of this bot, use ibb!getbotroles to get your dev roles",
					},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("error sending embed to bot logs channel, but successfully certified bot: %w", err)
		}
	} else {
		// Denial is always possible
		return nil
	}

	return nil
}

func reviewLogicStaff(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	userId, err := snowflake.Parse(resp.UserID)

	if err != nil {
		return fmt.Errorf("error parsing user ID: %w", err)
	}

	if approve {
		err := state.Discord.Rest().AddMemberRole(state.Config.Servers.Main, userId, state.Config.Roles.AwaitingStaff)

		if err != nil {
			return err
		}

		// DM the user
		dmchan, err := state.Discord.Rest().CreateDMChannel(userId)

		if err != nil {
			return errors.New("could not send DM, please ask the user to accept DMs from server members")
		}

		if len(reason) > 1024 {
			return errors.New("reason must be 1024 characters or less")
		}

		_, err = state.Discord.Rest().CreateMessage(dmchan.ID(), discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Staff Application Whitelisted",
					Description: "Your staff application has been whitelisted for onboarding! Please ping any manager at #staff-only in our discord server to get started.",
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Feedback",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "Congratulations!",
					},
				},
			},
		})

		if err != nil {
			return errors.New("could not send DM, please ask the user to accept DMs from server members")
		}

		return nil
	} else {
		if strings.HasPrefix(reason, "MANUALLYNOTIFIED ") {
			state.Logger.Info("forcing denial of staff application that was manually notified by a manager", zap.String("userID", resp.UserID))
			return nil
		}

		// Attempt to DM the user on denial
		dmchan, err := state.Discord.Rest().CreateDMChannel(userId)

		if err != nil {
			return fmt.Errorf("could not create DM channel with user, please inform them manually, then deny with reason of 'MANUALLYNOTIFIED <your reason here>': %w", err)
		}

		_, err = state.Discord.Rest().CreateMessage(dmchan.ID(), discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Staff Application Denied",
					Description: "Unfortunately, we have denied your staff application for Infinity Bot List. You may reapply later if you wish to",
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Reason",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "Better luck next time?",
					},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("could not send DM, please inform them manually, then deny with reason of 'MANUALLYNOTIFIED <your reason here>': %w", err)
		}

		return nil
	}
}
