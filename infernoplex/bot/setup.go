package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/infernoplex/dclient"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const setupConfirmDescription = `
The following setup will now be performed:

- A new team will be created for your server, owned by you. You can add other administrators to it afterward through ` + "`Team Settings`" + `.
- This server will be added and will be owned by the team. Note that you can transfer ownership of this team to anyone on Omniplex if you want to.
- The server created will be set as a ` + "`draft`" + ` and will not be visible until it is published.

Notes:
- If you wish to recover access to this server (rogue moderator/admin etc) within Omniplex, please contact [support](https://omniplex.gg/redirect/discord)
- **Please now prepare a short and long description for your server.** You can change these later through ` + "`Server Settings`" + ` on the website.
- By continuing, you agree that you have read and understood the [Terms of Service](https://omniplex.gg/legal/terms)
`

type setupSession struct {
	AuthorID  string
	GuildID   snowflake.ID
	ExpiresAt time.Time
}

var setupSessions = map[string]*setupSession{}

func cmdSetup() *Command {
	return &Command{
		Name:        "setup",
		Description: "Sets up a server, needs 'Administrator' permissions on the author",
		AdminOnly:   true,
		Run: func(c *Ctx) error {
			if c.GuildID == 0 {
				return errors.New("This command must be run in a server!")
			}

			serverID := c.GuildID.String()

			var count int64

			if err := state.Pool.QueryRow(c.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", serverID).Scan(&count); err != nil {
				return err
			}

			if count > 0 {
				url := fmt.Sprintf("%s/servers/%s", state.Config.Sites.Frontend.Parse(), serverID)

				return c.Send(discord.MessageCreate{
					Embeds: []discord.Embed{{
						Title:       "Server Already Setup",
						URL:         url,
						Description: "Currently, most server settings can only be changed from the website!",
					}},
					Components: []discord.ContainerComponent{
						discord.NewActionRow(discord.NewLinkButton("Redirect", url)),
					},
				})
			}

			sessionsMu.Lock()
			setupSessions[c.Author.ID.String()] = &setupSession{
				AuthorID:  c.Author.ID.String(),
				GuildID:   c.GuildID,
				ExpiresAt: time.Now().Add(wizardTTL),
			}
			sessionsMu.Unlock()

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Confirm Setup?",
					Description: setupConfirmDescription,
				}},
				Components: []discord.ContainerComponent{
					discord.NewActionRow(
						discord.NewPrimaryButton("Next", "setup:next"),
						discord.NewDangerButton("Cancel", "setup:cancel"),
					),
				},
			})
		},
	}
}

func handleSetupComponent(ctx context.Context, e *events.ComponentInteractionCreate, id string) {
	userID := e.User().ID.String()

	sessionsMu.Lock()
	s, ok := setupSessions[userID]
	sessionsMu.Unlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		return
	}

	logFailedEdit("clear setup confirm buttons", clearComponents(e.Channel().ID(), e.Message.ID))

	if id == "setup:cancel" {
		sessionsMu.Lock()
		delete(setupSessions, userID)
		sessionsMu.Unlock()

		logFailedEdit("answer setup cancel", e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{{Description: "Cancelled."}},
		}))

		return
	}

	modal := discord.ModalCreate{
		CustomID: "setup:info",
		Title:    "Initial Setup",
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewTextInput("vanity", discord.TextInputStyleShort, "Vanity").
					WithPlaceholder("This must be unique, so think hard!").
					WithMinLength(1).WithMaxLength(20).WithRequired(true),
			),
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

	logFailedEdit("open setup info modal", e.Modal(modal))
}

func handleSetupInfoModal(ctx context.Context, e *events.ModalSubmitInteractionCreate) {
	userID := e.User().ID.String()

	sessionsMu.Lock()
	s, ok := setupSessions[userID]
	sessionsMu.Unlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		return
	}

	vanity := e.Data.Text("vanity")
	short := e.Data.Text("short")
	long := e.Data.Text("long")
	guildID := s.GuildID

	sessionsMu.Lock()
	s.ExpiresAt = time.Now().Add(wizardTTL)
	sessionsMu.Unlock()

	c := ctxFromModal(ctx, e)

	err := startInviteWizard(c, func(ctx context.Context, c *Ctx, inviteStr string) error {
		sessionsMu.Lock()
		delete(setupSessions, userID)
		sessionsMu.Unlock()

		return finishSetup(ctx, c, guildID, vanity, short, long, inviteStr)
	})

	if err != nil {
		state.Logger.Error("Failed to open invite wizard from setup", zap.Error(err))
	}
}

func finishSetup(ctx context.Context, c *Ctx, guildID snowflake.ID, vanity, short, long, inviteStr string) error {
	if err := c.Defer(); err != nil {
		return err
	}

	guild, err := dclient.Get().Rest().GetGuild(guildID, true)

	if err != nil {
		return fmt.Errorf("failed to fetch server info: %w", err)
	}

	ownerID := guild.OwnerID.String()
	authorID := c.Author.ID.String()

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	teamID := uuid.New()
	teamVanity := crypto.RandString(32)

	var teamVanityRef pgtype.UUID

	if err := tx.QueryRow(ctx,
		"INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, 'team') RETURNING itag",
		teamVanity, teamID.String(),
	).Scan(&teamVanityRef); err != nil {
		return fmt.Errorf("error while creating team vanity: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'infernoplex')",
		teamID, guild.Name+"'s Team", teamVanityRef,
	); err != nil {
		return fmt.Errorf("error while creating team: %w", err)
	}

	if err := ensureUserExists(ctx, tx, ownerID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'infernoplex')",
		teamID, ownerID, []string{perms.EntityOwner.String()},
	); err != nil {
		return fmt.Errorf("error while adding team owner: %w", err)
	}

	if authorID != ownerID {
		if err := ensureUserExists(ctx, tx, authorID); err != nil {
			return err
		}

		coAdminPerms := []string{
			perms.EntityEditServers.String(),
			perms.EntityDeleteServers.String(),
			perms.EntityManageServerInvites.String(),
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'infernoplex')",
			teamID, authorID, coAdminPerms,
		); err != nil {
			return fmt.Errorf("error while adding co-admin: %w", err)
		}
	}

	var vanityCount int64

	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM vanity WHERE code::text = $1", vanity).Scan(&vanityCount); err != nil {
		return fmt.Errorf("error while checking vanity: %w", err)
	}

	if vanityCount > 0 {
		return errors.New("Vanity already exists. Please rerun `/setup` with a new vanity!")
	}

	var serverVanityRef pgtype.UUID

	if err := tx.QueryRow(ctx,
		"INSERT INTO vanity (code, target_id, target_type) VALUES ($1::text, $2, 'server') RETURNING itag",
		vanity, guildID.String(),
	).Scan(&serverVanityRef); err != nil {
		return fmt.Errorf("error while creating server vanity: %w", err)
	}

	avatar := ""

	if iconURL := guild.IconURL(); iconURL != nil {
		avatar = *iconURL
	}

	if _, err := tx.Exec(ctx, `INSERT INTO servers (
			server_id, name, team_owner, total_members, online_members,
			short, long, invite, vanity_ref, extra_links, nsfw, avatar
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		guildID.String(), guild.Name, teamID, guild.ApproximateMemberCount, guild.ApproximatePresenceCount,
		short, long, inviteStr, serverVanityRef, []types.Link{}, guild.NSFWLevel == discord.NSFWLevelExplicit, avatar,
	); err != nil {
		return fmt.Errorf("error while creating server: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return c.Ok("All done :white_check_mark:")
}

func ensureUserExists(ctx context.Context, tx pgx.Tx, userID string) error {
	var count int64

	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE user_id = $1", userID).Scan(&count); err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}

	if count > 0 {
		return nil
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO users (user_id, extra_links, developer, certified) VALUES ($1, $2, false, false)",
		userID, []types.Link{},
	); err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	return nil
}
