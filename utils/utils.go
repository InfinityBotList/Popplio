package utils

import (
	"context"
	"encoding/json"
	"math/rand"
	"reflect"
	"time"
	"unsafe"

	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

func IsNone(s string) bool {
	if s == "None" || s == "none" || s == "" || s == "null" {
		return true
	}
	return false
}

func ParseBot(bot *types.Bot) {
	if IsNone(bot.Website.String) {
		bot.Website.Status = pgtype.Null
	}

	if IsNone(bot.Donate.String) {
		bot.Donate.Status = pgtype.Null
	}

	if IsNone(bot.Github.String) {
		bot.Github.Status = pgtype.Null
	}

	if IsNone(bot.Support.String) {
		bot.Support.Status = pgtype.Null
	}

	if IsNone(bot.Banner.String) {
		bot.Banner.Status = pgtype.Null
	}

	if IsNone(bot.Invite.String) {
		bot.Invite.Status = pgtype.Null
	}
}

func ResolveBotPack(ctx context.Context, pool *pgxpool.Pool, pack *types.BotPack, s *discordgo.Session, redisCache *redis.Client) error {
	ownerUser, err := GetDiscordUser(s, redisCache, ctx, pack.Owner)

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
		var certified bool
		var shards int
		var votes int
		var invites int
		var servers int
		var tags []string
		err := pool.QueryRow(ctx, "SELECT short, type, vanity, banner, nsfw, premium, certified, shards, votes, invites, servers, tags FROM bots WHERE bot_id = $1", botId).Scan(&short, &bot_type, &vanity, &banner, &nsfw, &premium, &certified, &shards, &votes, &invites, &servers, &tags)

		if err == pgx.ErrNoRows {
			continue
		}

		if err != nil {
			return err
		}

		botUser, err := GetDiscordUser(s, redisCache, ctx, botId)

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
			Certified:    certified,
			Shards:       shards,
			Votes:        votes,
			InviteClicks: invites,
			Servers:      servers,
			Tags:         tags,
		})
	}

	return nil
}

func ParseUser(user *types.User) {
	if IsNone(user.Website.String) {
		user.Website.Status = pgtype.Null
	}

	if IsNone(user.Github.String) {
		user.Github.Status = pgtype.Null
	}

	if IsNone(user.About.String) {
		user.About.Status = pgtype.Null
	}

	if IsNone(user.Nickname.String) {
		user.Nickname.Status = pgtype.Null
	}
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

func GetDiscordUser(s *discordgo.Session, redisCache *redis.Client, ctx context.Context, id string) (*types.DiscordUser, error) {
	// Check if in discordgo session first

	const userExpiryTime = 4 * time.Hour

	if s.State != nil {
		guilds := s.State.Guilds

		for _, guild := range guilds {
			member, err := s.State.Member(guild.ID, id)

			if err == nil {
				p, pErr := s.State.Presence(guild.ID, id)

				if pErr != nil {
					p = &discordgo.Presence{
						User:   member.User,
						Status: discordgo.StatusOffline,
					}
				}

				obj := &types.DiscordUser{
					ID:            id,
					Username:      member.User.Username,
					Avatar:        member.User.AvatarURL(""),
					Discriminator: member.User.Discriminator,
					Bot:           member.User.Bot,
					Nickname:      member.Nick,
					Guild:         guild.ID,
					Mention:       member.User.Mention(),
					Status:        p.Status,
					System:        member.User.System,
					Flags:         member.User.Flags,
					Tag:           member.User.Username + "#" + member.User.Discriminator,
				}

				bytes, err := json.Marshal(obj)

				if err == nil {
					redisCache.Set(ctx, "uobj:"+id, bytes, userExpiryTime)
				}

				return obj, nil
			}
		}
	}

	// Check if in redis cache
	userBytes, err := redisCache.Get(ctx, "uobj:"+id).Result()

	if err == nil {
		// Try to unmarshal

		var user types.DiscordUser

		err = json.Unmarshal([]byte(userBytes), &user)

		if err == nil {
			return &user, err
		}
	}

	// Get from discord
	user, err := s.User(id)

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
	redisCache.Set(ctx, "uobj:"+id, obj, userExpiryTime)

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

func GetVoteData(ctx context.Context, pool *pgxpool.Pool, userID, botID string) (*types.UserVote, error) {
	var votes []int64

	var voteDates []*struct {
		Date pgtype.Timestamptz `db:"date"`
	}

	rows, err := pool.Query(ctx, "SELECT date FROM votes WHERE user_id = $1 AND bot_id = $2 ORDER BY date DESC", userID, botID)

	if err != nil {
		return nil, err
	}

	err = pgxscan.ScanAll(&voteDates, rows)

	for _, vote := range voteDates {
		if vote.Date.Status != pgtype.Null {
			votes = append(votes, vote.Date.Time.UnixMilli())
		}
	}

	voteParsed := types.UserVote{
		VoteTime: GetVoteTime(),
	}

	log.WithFields(log.Fields{
		"uid":   userID,
		"bid":   botID,
		"votes": votes,
		"err":   err,
	}).Info("Got vote info")

	voteParsed.Timestamps = votes

	// In most cases, will be one but not always
	if len(votes) > 0 {
		if time.Now().UnixMilli() < votes[0] {
			log.Error("detected illegal vote time", votes[0])
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

		if db == "-" || db == "" {
			continue
		}

		cols = append(cols, db)
	}

	return cols
}
