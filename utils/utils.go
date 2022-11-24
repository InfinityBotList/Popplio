package utils

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"popplio/state"
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	userBotColsArr = GetCols(types.UserBot{})
	// These are the columns of a userbot object
	userBotCols = strings.Join(userBotColsArr, ",")
)

func IsNone(s string) bool {
	if s == "None" || s == "none" || s == "" || s == "null" {
		return true
	}
	return false
}

func ParseBot(ctx context.Context, pool *pgxpool.Pool, bot *types.Bot, s *discordgo.Session, redisCache *redis.Client) error {
	if IsNone(bot.Banner.String) || !strings.HasPrefix(bot.Banner.String, "https://") {
		bot.Banner.Valid = false
		bot.Banner.String = ""
	}

	if IsNone(bot.Invite.String) || !strings.HasPrefix(bot.Invite.String, "https://") {
		bot.Invite.Valid = false
		bot.Invite.String = ""
	}

	ownerUser, err := GetDiscordUser(bot.Owner)

	if err != nil {
		return err
	}

	bot.MainOwner = ownerUser

	botUser, err := GetDiscordUser(bot.BotID)

	if err != nil {
		return err
	}

	bot.User = botUser

	bot.ResolvedAdditionalOwners = []*types.DiscordUser{}

	for _, owner := range bot.AdditionalOwners {
		ownerUser, err := GetDiscordUser(owner)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		bot.ResolvedAdditionalOwners = append(bot.ResolvedAdditionalOwners, ownerUser)
	}

	bot.LongDescIsURL = strings.HasPrefix(strings.ReplaceAll(bot.Long, "\n", ""), "https://")

	if bot.LongDescIsURL {
		/*
		   desc = `<iframe src="${bot.long
		       .replace('\n', '')
		       .replace(
		           ' ',
		           ''
		       )}" width="100%" height="100%" style="width: 100%; height: 100vh; color: black;"><object data="${bot.long
		       .replace('\n', '')
		       .replace(
		           ' ',
		           ''
		       )}" width="100%" height="100%" style="width: 100%; height: 100vh; color: black;"><embed src="${bot.long
		       .replace('\n', '')
		       .replace(
		           ' ',
		           ''
		       )}" width="100%" height="100%" style="width: 100%; height: 100vh; color: black;"> </embed>${bot.long
		       .replace('\n', '')
		       .replace(' ', '')}</object></iframe>`
		*/

		longDesc := strings.ReplaceAll(bot.Long, "\n", "")

		bot.Long = "<iframe src=\"" + longDesc + "\" width=\"100%\" height=\"100%\" style=\"width: 100%; height: 100vh; color: black;\"><object data=\"" + longDesc + "\" width=\"100%\" height=\"100%\" style=\"width: 100%; height: 100vh; color: black;\"><embed src=\"" + longDesc + "\" width=\"100%\" height=\"100%\" style=\"width: 100%; height: 100vh; color: black;\">i</embed>" + longDesc + "</object></iframe>"
	}

	return nil
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
	rows, err := state.Pool.Query(ctx, "SELECT user_id, upvote, date FROM pack_votes WHERE url = $1", url)

	if err != nil {
		return []types.PackVote{}, err
	}

	defer rows.Close()

	votes := []types.PackVote{}

	for rows.Next() {
		// Fetch votes for the pack
		var userId string
		var upvote bool
		var date time.Time

		err := rows.Scan(&userId, &upvote, &date)

		if err != nil {
			return nil, err
		}

		votes = append(votes, types.PackVote{
			UserID: userId,
			Upvote: upvote,
			Date:   date,
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

func ParseUser(ctx context.Context, pool *pgxpool.Pool, user *types.User, s *discordgo.Session, redisCache *redis.Client) error {
	if IsNone(user.About.String) {
		user.About.Valid = false
		user.About.String = ""
	}

	userObj, err := GetDiscordUser(user.ID)

	if err != nil {
		return err
	}

	user.User = userObj

	userBotsRows, err := pool.Query(ctx, "SELECT "+userBotCols+" FROM bots WHERE owner = $1 OR additional_owners && $2", user.ID, []string{user.ID})

	if err != nil {
		return err
	}

	var userBots []types.UserBot = []types.UserBot{}

	err = pgxscan.ScanAll(&userBots, userBotsRows)

	if err != nil {
		return err
	}

	parsedUserBots := []types.UserBot{}
	for _, bot := range userBots {
		userObj, err := GetDiscordUser(bot.BotID)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		bot.User = userObj
		parsedUserBots = append(parsedUserBots, bot)
	}

	user.UserBots = parsedUserBots

	return nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func RandString(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
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
		Nickname:      "Member not found",
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
		Date pgtype.Timestamptz `db:"date"`
	}

	rows, err := state.Pool.Query(ctx, "SELECT date FROM votes WHERE user_id = $1 AND bot_id = $2 ORDER BY date DESC", userID, botID)

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

		cols = append(cols, db)
	}

	return cols
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
