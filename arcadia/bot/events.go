package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/impls"
	"popplio/config"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

// onGuildsReady runs the startup fixups the Rust Ready handler did.
//
// ORDERING (§14c): upstream started the panel API from inside the Ready handler
// so the Discord cache was warm before the API accepted traffic. Here Popplio
// owns the Discord connection and starts the panel from main, so the panel can
// come up before the cache fills. The paths that read the cache (dovewing tier 1,
// role validation in staff positions, member checks in RPC) all already treat an
// uncached guild as "not found" and degrade rather than fail, and dovewing falls
// through to Postgres. See CONFORMANCE.md.
func onGuildsReady(ctx context.Context, _ *events.GuildsReady) {
	self, ok := state.Discord.Caches().SelfUser()

	name := "arcadia"
	if ok {
		name = self.Username
	}

	state.Logger.Info(fmt.Sprintf("%s is ready! Doing some minor DB fixes", name))

	_, err := state.Pool.Exec(ctx, "UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE LOWER(claimed_by) = 'none'")

	if err != nil {
		state.Logger.Error("Failed to run startup DB fixes", zap.Error(err))
	}

	if err := SyncCommands(); err != nil {
		state.Logger.Error("Failed to sync application commands", zap.Error(err))
	}
}

// onGuildMemberJoin announces new members and bots in the main guild.
//
// The whole handler is a no-op on staging.
func onGuildMemberJoin(e *events.GuildMemberJoin) {
	if config.CurrentEnv == config.CurrentEnvStaging {
		return
	}

	if e.GuildID != state.Config.Servers.Main {
		return
	}

	member := e.Member
	face := member.User.EffectiveAvatarURL()
	now := time.Now()

	if member.User.Bot {
		err := impls.SendChannel(state.Config.Channels.System, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:     "__**New Bot Added**__",
				Color:     impls.ColourGreen,
				Thumbnail: &discord.EmbedResource{URL: face},
				Timestamp: &now,
				Description: fmt.Sprintf(
					"Bot <@%s> (%s) has been added to the server.", member.User.ID, member.User.Username),
			}},
		})

		if err != nil {
			state.Logger.Error("Failed to announce new bot", zap.Error(err))
		}

		err = impls.AddRole(state.Config.Servers.Main, member.User.ID, state.Config.Roles.BotRole, "Bot added to server")

		if err != nil {
			state.Logger.Error("Failed to add bot role", zap.Error(err))
		}

		return
	}

	err := impls.SendChannel(state.Config.Channels.System, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:     "__**New User**__",
			Color:     impls.ColourGreen,
			Thumbnail: &discord.EmbedResource{URL: face},
			Timestamp: &now,
			Description: fmt.Sprintf(
				"Hmmmm... looks like <@%s> (%s) has strolled in. Can we trust them?", member.User.ID, member.User.Username),
		}},
	})

	if err != nil {
		state.Logger.Error("Failed to announce new user", zap.Error(err))
	}
}

// --- Command guards (§11.4) ---

// mainServer requires the command to be run in the main guild.
func mainServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Main {
		return errors.New("You are not in the main server")
	}

	return nil
}

// staffServer requires the command to be run in the staff guild.
func staffServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Staff {
		return errors.New("You are not in the staff server")
	}

	return nil
}

// testingServer requires the command to be run in the testing guild.
func testingServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Testing {
		return errors.New("You are not in the testing server")
	}

	return nil
}

// isStaff ERRORS rather than returning false, which is how the message reaches
// the user.
func isStaff(c *Ctx) error {
	var count int64

	err := state.Pool.QueryRow(c.Context, "SELECT COUNT(*) FROM staff_members WHERE user_id = $1", c.Author.ID.String()).Scan(&count)

	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("You are not staff")
	}

	return nil
}
