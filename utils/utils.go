package utils

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"popplio/constants"
	"popplio/state"
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Returns if a string is empty/null or not. Used throughout the codebase
func IsNone(s string) bool {
	if s == "None" || s == "none" || s == "" || s == "null" {
		return true
	}
	return false
}

// Returns the votes of a pack, Used throughout the codebase
func ResolvePackVotes(ctx context.Context, url string) ([]types.PackVote, error) {
	rows, err := state.Pool.Query(ctx, "SELECT user_id, upvote, created_at FROM pack_votes WHERE url = $1", url)

	if err != nil {
		return []types.PackVote{}, err
	}

	defer rows.Close()

	votes := []types.PackVote{}

	for rows.Next() {
		// Fetch votes for the pack
		var userId string
		var upvote bool
		var createdAt time.Time

		err := rows.Scan(&userId, &upvote, &createdAt)

		if err != nil {
			return nil, err
		}

		votes = append(votes, types.PackVote{
			UserID:    userId,
			Upvote:    upvote,
			CreatedAt: createdAt,
		})
	}

	return votes, nil
}

func GetDiscordUser(id string) (*types.DiscordUser, error) {
	// Check if in discordgo session first

	const userExpiryTime = 8 * time.Hour

	// Before wasting time searching state, ensure the ID is actually a valid snowflake
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return nil, err
	}

	// For all practical purposes, a simple length check can handle a lot of illegal IDs
	if len(id) <= 16 || len(id) > 20 {
		return nil, errors.New("invalid snowflake")
	}

	if state.Discord.State != nil {
		guilds := state.Discord.State.Guilds

		// First try for main server
		member, err := state.Discord.State.Member(state.Config.Servers.Main, id)

		if err == nil {
			p, pErr := state.Discord.State.Presence(state.Config.Servers.Main, id)

			if pErr != nil {
				p = &discordgo.Presence{
					User:   member.User,
					Status: discordgo.StatusOffline,
				}
			}

			if member.User.Bot {
				_, err = state.Pool.Exec(state.Context, "UPDATE bots SET queue_name = $1, queue_avatar = $2 WHERE bot_id = $3", member.User.Username, member.User.AvatarURL(""), member.User.ID)

				if err != nil {
					state.Logger.Error("Failed to update bot queue name", zap.Error(err))
				}
			} else {
				_, err = state.Pool.Exec(state.Context, "UPDATE users SET username = $1, avatar = $2 WHERE user_id = $3", member.User.Username, member.User.AvatarURL(""), member.User.ID)

				if err != nil {
					state.Logger.Error("Failed to update bot queue name", zap.Error(err))
				}
			}

			obj := &types.DiscordUser{
				ID:             id,
				Username:       member.User.Username,
				Avatar:         member.User.AvatarURL(""),
				Discriminator:  member.User.Discriminator,
				Bot:            member.User.Bot,
				Nickname:       member.Nick,
				Guild:          state.Config.Servers.Main,
				Mention:        member.User.Mention(),
				Status:         p.Status,
				System:         member.User.System,
				Flags:          member.User.Flags,
				Tag:            member.User.Username + "#" + member.User.Discriminator,
				IsServerMember: true,
			}

			bytes, err := json.Marshal(obj)

			if err == nil {
				state.Redis.Set(state.Context, "uobj:"+id, bytes, userExpiryTime)
			}

			return obj, nil
		}

		for _, guild := range guilds {
			if guild.ID == state.Config.Servers.Main {
				continue // Already checked
			}

			member, err := state.Discord.State.Member(guild.ID, id)

			if err == nil {
				p, pErr := state.Discord.State.Presence(guild.ID, id)

				if pErr != nil {
					p = &discordgo.Presence{
						User:   member.User,
						Status: discordgo.StatusOffline,
					}
				}

				if member.User.Bot {
					_, err = state.Pool.Exec(state.Context, "UPDATE bots SET queue_name = $1, queue_avatar = $2 WHERE bot_id = $3", member.User.Username, member.User.AvatarURL(""), member.User.ID)

					if err != nil {
						state.Logger.Error("Failed to update bot queue name", zap.Error(err))
					}
				} else {
					_, err = state.Pool.Exec(state.Context, "UPDATE users SET username = $1, avatar = $2 WHERE user_id = $3", member.User.Username, member.User.AvatarURL(""), member.User.ID)

					if err != nil {
						state.Logger.Error("Failed to update bot queue name", zap.Error(err))
					}
				}

				obj := &types.DiscordUser{
					ID:             id,
					Username:       member.User.Username,
					Avatar:         member.User.AvatarURL(""),
					Discriminator:  member.User.Discriminator,
					Bot:            member.User.Bot,
					Nickname:       member.Nick,
					Guild:          guild.ID,
					Mention:        member.User.Mention(),
					Status:         p.Status,
					System:         member.User.System,
					Flags:          member.User.Flags,
					Tag:            member.User.Username + "#" + member.User.Discriminator,
					IsServerMember: false,
				}

				bytes, err := json.Marshal(obj)

				if err == nil {
					state.Redis.Set(state.Context, "uobj:"+id, bytes, userExpiryTime)
				}

				return obj, nil
			}
		}
	}

	// Check if in redis cache
	userBytes, err := state.Redis.Get(state.Context, "uobj:"+id).Result()

	if err == nil {
		// Try to unmarshal

		var user types.DiscordUser

		err = json.Unmarshal([]byte(userBytes), &user)

		if err == nil {
			return &user, err
		}
	}

	// Get from discord
	user, err := state.Discord.User(id)

	if err != nil {
		return nil, err
	}

	if user.Bot {
		_, err = state.Pool.Exec(state.Context, "UPDATE bots SET queue_name = $1, queue_avatar = $2 WHERE bot_id = $3", user.Username, user.AvatarURL(""), user.ID)

		if err != nil {
			state.Logger.Error("Failed to update bot queue name", zap.Error(err))
		}
	} else {
		_, err = state.Pool.Exec(state.Context, "UPDATE users SET username = $1, avatar = $2 WHERE user_id = $3", user.Username, user.AvatarURL(""), user.ID)

		if err != nil {
			state.Logger.Error("Failed to update bot queue name", zap.Error(err))
		}
	}

	obj := &types.DiscordUser{
		ID:            id,
		Username:      user.Username,
		Avatar:        user.AvatarURL(""),
		Discriminator: user.Discriminator,
		Bot:           user.Bot,
		Nickname:      "",
		Guild:         "",
		Mention:       user.Mention(),
		Status:        discordgo.StatusOffline,
		System:        user.System,
		Flags:         user.Flags,
		Tag:           user.Username + "#" + user.Discriminator,
	}

	// Store in redis
	state.Redis.Set(state.Context, "uobj:"+id, obj, userExpiryTime)

	return obj, nil
}

func GetDatabaseDiscordUser(ctx context.Context, id string) (*types.DatabaseDiscordUser, error) {
	var count int64

	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE user_id = $1", id).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return nil, err
	}

	if count == 0 {
		return &types.DatabaseDiscordUser{
			FoundInDB: false,
		}, nil
	}

	var username string
	var avatar string

	err = state.Pool.QueryRow(ctx, "SELECT username, avatar FROM users WHERE user_id = $1", id).Scan(&username, &avatar)

	if err != nil {
		state.Logger.Error(err)
		return nil, err
	}

	if avatar == "unset" {
		user, err := GetDiscordUser(id)

		if err != nil {
			state.Logger.Error(err)
			return nil, err
		}

		avatar = user.Avatar
	}

	return &types.DatabaseDiscordUser{
		ID:        id,
		Username:  username,
		Avatar:    avatar,
		FoundInDB: true,
	}, nil
}

func GetDoubleVote() bool {
	return time.Now().Weekday() == time.Friday || time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday
}

func GetVoteTime() uint16 {
	if GetDoubleVote() {
		return 6
	} else {
		return 12
	}
}

func GetVoteData(ctx context.Context, userID, botID string, log bool) (*types.UserVote, error) {
	var votes []int64

	var voteDates []*struct {
		Date pgtype.Timestamptz `db:"created_at"`
	}

	rows, err := state.Pool.Query(ctx, "SELECT created_at FROM votes WHERE user_id = $1 AND bot_id = $2 ORDER BY created_at DESC", userID, botID)

	if err != nil {
		return nil, err
	}

	err = pgxscan.ScanAll(&voteDates, rows)

	for _, vote := range voteDates {
		if vote.Date.Valid {
			votes = append(votes, vote.Date.Time.UnixMilli())
		}
	}

	voteParsed := types.UserVote{
		UserID: userID,
		VoteInfo: types.VoteInfo{
			Weekend:  GetDoubleVote(),
			VoteTime: GetVoteTime(),
		},
	}

	if log {
		state.Logger.With(
			zap.String("user_id", userID),
			zap.String("bot_id", botID),
			zap.Int64s("votes", votes),
			zap.Error(err),
		).Info("Got vote data")
	}

	voteParsed.Timestamps = votes

	// In most cases, will be one but not always
	if len(votes) > 0 {
		if time.Now().UnixMilli() < votes[0] {
			state.Logger.Error("detected illegal vote time", votes[0])
			votes[0] = time.Now().UnixMilli()
		}

		if time.Now().UnixMilli()-votes[0] < int64(GetVoteTime())*60*60*1000 {
			voteParsed.HasVoted = true
			voteParsed.LastVoteTime = votes[0]
		}
	}

	if voteParsed.LastVoteTime == 0 && len(votes) > 0 {
		voteParsed.LastVoteTime = votes[0]
	}
	return &voteParsed, nil
}

func GetCols(s any) []string {
	refType := reflect.TypeOf(s)

	var cols []string

	for _, f := range reflect.VisibleFields(refType) {
		db := f.Tag.Get("db")
		reflectOpts := f.Tag.Get("reflect")

		if db == "-" || db == "" || reflectOpts == "ignore" {
			continue
		}

		// Do not allow even accidental fetches of tokens
		if db == "api_token" || db == "webhook_secret" {
			continue
		}

		cols = append(cols, db)
	}

	return cols
}

// Returns true if the user is a owner of the bot
func IsBotOwner(ctx context.Context, userID string, botID string) (bool, error) {
	// Validate that they actually own this bot
	var count int64
	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE bot_id = $1 AND (owner = $2 OR additional_owners && $3)", botID, userID, []string{userID}).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error getting ownership info: %v", err)
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func ClearBotCache(ctx context.Context, botId string) error {
	// Get name and vanity, delete from cache
	var vanity string
	var clientId string

	err := state.Pool.QueryRow(ctx, "SELECT lower(vanity), client_id FROM bots WHERE bot_id = $1", botId).Scan(&vanity, &clientId)

	if err != nil {
		return err
	}

	// Delete from cache
	state.Redis.Del(ctx, "bc-"+vanity)
	state.Redis.Del(ctx, "bc-"+botId)
	state.Redis.Del(ctx, "bc-"+clientId)

	return nil
}

func ValidateExtraLinks(links []types.Link) error {
	var public, private int

	if len(links) > 20 {
		return errors.New("you have too many links")
	}

	for _, link := range links {
		if strings.HasPrefix(link.Name, "_") {
			private++

			if len(link.Name) > 512 || len(link.Value) > 8192 {
				return errors.New("one of your private links has a name/value that is too long")
			}

			if strings.ReplaceAll(link.Name, " ", "") == "" || strings.ReplaceAll(link.Value, " ", "") == "" {
				return errors.New("one of your private links has a name/value that is empty")
			}
		} else {
			public++

			if len(link.Name) > 64 || len(link.Value) > 512 {
				return errors.New("one of your public links has a name/value that is too long")
			}

			if strings.ReplaceAll(link.Name, " ", "") == "" || strings.ReplaceAll(link.Value, " ", "") == "" {
				return errors.New("one of your public links has a name/value that is empty")
			}

			if !strings.HasPrefix(link.Value, "https://") {
				return errors.New("extra link '" + link.Name + "' must be HTTPS")
			}
		}

		for _, ch := range link.Name {
			allowedChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_ "

			if !strings.ContainsRune(allowedChars, ch) {
				return errors.New("extra link '" + link.Name + "' has an invalid character: " + string(ch))
			}
		}
	}

	if public > 10 {
		return errors.New("you have too many public links")
	}

	if private > 10 {
		return errors.New("you have too many private links")
	}

	return nil
}

func ResolveBot(ctx context.Context, name string) (string, error) {
	// First check count so we can avoid expensive DB calls
	var count int64

	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE "+constants.ResolveBotSQL, name).Scan(&count)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", nil
	}

	if count > 1 {
		// Delete one of the bots
		_, err := state.Pool.Exec(ctx, "DELETE FROM bots WHERE "+constants.ResolveBotSQL+" LIMIT 1", name)

		if err != nil {
			return "", err
		}
	}

	var id string
	err = state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE "+constants.ResolveBotSQL, name).Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}
