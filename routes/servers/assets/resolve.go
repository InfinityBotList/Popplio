// Package assets holds the server logic shared between endpoints.
package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var inviteCodeRegex = regexp.MustCompile(`(?:discord(?:\.gg|\.com/invite|app\.com/invite)/)?([a-zA-Z0-9-]+)/?$`)

// ResolveInvite extracts the invite code from a raw invite link/code and
// resolves it against Discord, returning the guild the invite points to.
// Shared by Add Server and Get Server Meta so the two never drift on what
// counts as a valid invite.
func ResolveInvite(ctx context.Context, rawInvite string) (*discord.Invite, error) {
	match := inviteCodeRegex.FindStringSubmatch(strings.TrimSpace(rawInvite))

	if len(match) < 2 || match[1] == "" {
		return nil, errors.New("could not find an invite code in that URL")
	}

	invite, err := state.Discord.Rest().GetInvite(match[1])

	if err != nil {
		return nil, fmt.Errorf("that invite is invalid, expired, or the server has revoked it: %w", err)
	}

	if invite.Guild == nil {
		return nil, errors.New("that invite is not for a server")
	}

	return invite, nil
}

// GuildIconURL returns the guild's icon CDN URL, or "" if it has none set.
func GuildIconURL(guild *discord.InviteGuild) string {
	if guild.Icon == nil {
		return ""
	}
	return discord.GuildIcon.URL(discord.FileFormatPNG, nil, guild.ID, *guild.Icon)
}

// BotGuildPresence is the result of asking Infernoplex whether it's
// currently a member of a guild.
type BotGuildPresence struct {
	Present   bool
	InviteURL string
}

// CheckBotGuildPresence asks Infernoplex (the tracking bot, a separate
// service from Popplio's own Discord client) whether it's currently a
// member of the given guild via its Sorbet API. Several features
// (emoji/sticker sync, real invite creation, live member counts) silently
// never work for a server unless the tracking bot is actually in it, so
// this is used to nudge owners to invite it during Add Server.
//
// Best-effort: any failure to reach Infernoplex is treated as "not present,
// no invite URL available" rather than failing the caller — this only
// drives a UI hint, nothing load-bearing depends on it.
func CheckBotGuildPresence(ctx context.Context, guildID string) BotGuildPresence {
	reqBody, err := json.Marshal(map[string]any{
		"IsInGuild": map[string]any{
			"guild_id": guildID,
		},
	})

	if err != nil {
		return BotGuildPresence{}
	}

	cli := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "POST", state.Config.Sites.Infernoplex.Parse()+"/", bytes.NewReader(reqBody))

	if err != nil {
		return BotGuildPresence{}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := cli.Do(req)

	if err != nil {
		state.Logger.Warn("Error while checking Infernoplex guild membership", zap.Error(err), zap.String("guildID", guildID))
		return BotGuildPresence{}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BotGuildPresence{}
	}

	var result struct {
		IsInGuild struct {
			InGuild   bool   `json:"in_guild"`
			InviteURL string `json:"invite_url"`
		} `json:"IsInGuild"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return BotGuildPresence{}
	}

	return BotGuildPresence{
		Present:   result.IsInGuild.InGuild,
		InviteURL: result.IsInGuild.InviteURL,
	}
}

func ResolveIndexServer(ctx context.Context, server *types.IndexServer) error {
	var code string

	err := state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", server.VanityRef).Scan(&code)

	if err != nil {
		return fmt.Errorf("error querying vanity table: %w", err)
	}

	server.Vanity = code

	server.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, server.ServerID, "server")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}

// ResolveIndexServers resolves every server in the slice concurrently, since
// each server's resolution (vanity lookup, vote count) is independent of
// every other server's. Returns the first error encountered, if any.
func ResolveIndexServers(ctx context.Context, servers []types.IndexServer) error {
	g, ctx := errgroup.WithContext(ctx)

	for i := range servers {
		g.Go(func() error {
			if err := ResolveIndexServer(ctx, &servers[i]); err != nil {
				return fmt.Errorf("serverID=%s: %w", servers[i].ServerID, err)
			}
			return nil
		})
	}

	return g.Wait()
}
