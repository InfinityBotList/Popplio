package votes

import (
	"context"
	"errors"
	"fmt"
	"popplio/assetmanager"
	"popplio/state"
	"strconv"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
)

type EntityInfo struct {
	Name    string
	URL     string
	VoteURL string
	Avatar  string
}

// GetEntityInfo returns information about the entity that is being voted for including vote bans etc.
//
// TODO: Refactor vote ban checks to its own function
func GetEntityInfo(ctx context.Context, targetId, targetType string) (*EntityInfo, error) {
	// Handle entity specific checks here, such as ensuring the entity actually exists
	switch targetType {
	case "bot":
		var botType string
		var voteBanned bool

		err := state.Pool.QueryRow(ctx, "SELECT type, vote_banned FROM bots WHERE bot_id = $1", targetId).Scan(&botType, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bot not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch bot data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("bot is vote banned and cannot be voted for right now")
		}

		if botType != "approved" && botType != "certified" {
			return nil, errors.New("bot is not approved or certified and cannot be voted for right now")
		}

		botObj, err := dovewing.GetUser(ctx, targetId, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, err
		}

		// Set entityInfo for log
		return &EntityInfo{
			URL:     "https://botlist.site/" + targetId,
			VoteURL: "https://botlist.site/" + targetId + "/vote",
			Name:    botObj.Username,
			Avatar:  botObj.Avatar,
		}, nil
	case "pack":
		return &EntityInfo{
			URL:     "https://botlist.site/pack/" + targetId,
			VoteURL: "https://botlist.site/pack/" + targetId,
			Name:    targetId,
			Avatar:  "",
		}, nil
	case "team":
		var name string
		var voteBanned bool

		err := state.Pool.QueryRow(ctx, "SELECT name, vote_banned FROM teams WHERE id = $1", targetId).Scan(&name, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("team not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch team data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("team is vote banned and cannot be voted for right now")
		}

		avatar := assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, targetId)

		var avatarPath string

		if avatar.Exists {
			avatarPath = state.Config.Sites.CDN + "/" + avatar.Path + "?ts=" + strconv.FormatInt(avatar.LastModified.Unix(), 10)
		} else {
			avatarPath = state.Config.Sites.CDN + "/" + avatar.DefaultPath
		}

		// Set entityInfo for log
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/team/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/team/" + targetId + "/vote",
			Name:    name,
			Avatar:  avatarPath,
		}, nil
	case "server":
		var name, avatar string
		var voteBanned bool

		err := state.Pool.QueryRow(ctx, "SELECT name, avatar, vote_banned FROM servers WHERE server_id = $1", targetId).Scan(&name, &avatar, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch server data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("server is vote banned and cannot be voted for right now")
		}

   		// Set entityInfo for log
                return &EntityInfo{
			URL: state.Config.Sites.Frontend.Parse()+ "/server/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/server/" + targetId + "/vote",
			Name:    name,
			Avatar:  avatar,
		}, nil
	case "blog":
	        return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/blog/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/blog/" + targetId,
			Name:    targetId,
			Avatar:  state.Config.Sites.CDN.Parse() + "/avatars/default.webp",
		}, nil
	default:
		return nil, errors.New("unimplemented target type:" + targetType)
	}
}
