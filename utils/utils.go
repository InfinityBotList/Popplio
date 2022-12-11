package utils

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func IsNone(s string) bool {
	if s == "None" || s == "none" || s == "" || s == "null" {
		return true
	}
	return false
}

func ResolveIndexBot(ib []types.IndexBot) ([]types.IndexBot, error) {
	for i, bot := range ib {
		botUser, err := GetDiscordUser(bot.BotID)

		if err != nil {
			return nil, err
		}

		ib[i].User = botUser
	}

	return ib, nil
}

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

func ResolveBotPack(ctx context.Context, pool *pgxpool.Pool, pack *types.BotPack) error {
	ownerUser, err := GetDiscordUser(pack.Owner)

	if err != nil {
		return err
	}

	pack.Votes, err = ResolvePackVotes(ctx, pack.URL)

	if err != nil {
		return err
	}

	pack.ResolvedOwner = ownerUser

	for _, botId := range pack.Bots {
		var short string
		var bot_type pgtype.Text
		var vanity pgtype.Text
		var banner pgtype.Text
		var nsfw bool
		var premium bool
		var shards int
		var votes int
		var inviteClicks int
		var servers int
		var tags []string
		err := pool.QueryRow(ctx, "SELECT short, type, vanity, banner, nsfw, premium, shards, votes, invite_clicks, servers, tags FROM bots WHERE bot_id = $1", botId).Scan(&short, &bot_type, &vanity, &banner, &nsfw, &premium, &shards, &votes, &inviteClicks, &servers, &tags)

		if err == pgx.ErrNoRows {
			continue
		}

		if err != nil {
			return err
		}

		botUser, err := GetDiscordUser(botId)

		if err != nil {
			return err
		}

		pack.ResolvedBots = append(pack.ResolvedBots, types.ResolvedPackBot{
			Short:        short,
			User:         botUser,
			Type:         bot_type,
			Vanity:       vanity,
			Banner:       banner,
			NSFW:         nsfw,
			Premium:      premium,
			Shards:       shards,
			Votes:        votes,
			InviteClicks: inviteClicks,
			Servers:      servers,
			Tags:         tags,
		})
	}

	return nil
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
		member, err := state.Discord.State.Member(os.Getenv("MAIN_SERVER"), id)

		if err == nil {
			p, pErr := state.Discord.State.Presence(os.Getenv("MAIN_SERVER"), id)

			if pErr != nil {
				p = &discordgo.Presence{
					User:   member.User,
					Status: discordgo.StatusOffline,
				}
			}

			obj := &types.DiscordUser{
				ID:             id,
				Username:       member.User.Username,
				Avatar:         member.User.AvatarURL(""),
				Discriminator:  member.User.Discriminator,
				Bot:            member.User.Bot,
				Nickname:       member.Nick,
				Guild:          os.Getenv("MAIN_SERVER"),
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
			if guild.ID == os.Getenv("MAIN_SERVER") {
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

func GetDoubleVote() bool {
	if time.Now().Weekday() == time.Friday || time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		return true
	} else {
		return false
	}
}

func GetVoteTime() uint16 {
	if GetDoubleVote() {
		return 6
	} else {
		return 12
	}
}

func GetVoteData(ctx context.Context, userID, botID string) (*types.UserVote, error) {
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
		VoteTime: GetVoteTime(),
	}

	state.Logger.With(
		zap.String("user_id", userID),
		zap.String("bot_id", botID),
		zap.Int64s("votes", votes),
		zap.Error(err),
	).Info("Got vote data")

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
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
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
