package votes

import (
	"context"
	"errors"
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
			state.Logger.Error(err)
			return nil, err
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

		err := state.Pool.QueryRow(ctx, "SELECT name FROM teams WHERE id = $1", targetId).Scan(&name)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("team not found")
		}

		if err != nil {
			return nil, err
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
			URL:     "https://botlist.site/team/" + targetId,
			VoteURL: "https://botlist.site/team/" + targetId + "/vote",
			Name:    name,
			Avatar:  avatarPath,
		}, nil
	case "server":
		var name, avatar string

		err := state.Pool.QueryRow(ctx, "SELECT name, avatar FROM servers WHERE server_id = $1", targetId).Scan(&name, &avatar)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, err
		}

		// Set entityInfo for log
		return &EntityInfo{
			URL:     "https://botlist.site/server/" + targetId,
			VoteURL: "https://botlist.site/server/" + targetId + "/vote",
			Name:    name,
			Avatar:  avatar,
		}, nil
	default:
		return nil, errors.New("unimplemented target type:" + targetType)
	}
}
