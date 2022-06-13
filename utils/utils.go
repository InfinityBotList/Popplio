package utils

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"time"
	"unsafe"

	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func IsNone(s *string) bool {
	if *s == "None" || *s == "none" || *s == "" || *s == "null" {
		return true
	}
	return false
}

func ParseBot(bot *types.Bot) {
	bot.Tags = strings.Split(strings.ReplaceAll(bot.TagsRaw, " ", ""), ",")

	if IsNone(bot.Website) {
		bot.Website = nil
	}

	if IsNone(bot.Donate) {
		bot.Donate = nil
	}

	if IsNone(bot.Github) {
		bot.Github = nil
	}
}

func ParseUser(user *types.User) {
	if IsNone(user.Website) {
		user.Website = nil
	}

	if IsNone(user.Github) {
		user.Github = nil
	}

	if IsNone(user.About) {
		user.About = nil
	}

	if IsNone(user.Nickname) {
		user.Nickname = nil
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

func GetVoteData(ctx context.Context, mongoDb *mongo.Database, userID, botID string) (*types.UserVote, error) {
	var votes []int64

	col := mongoDb.Collection("votes")

	findOptions := options.Find()

	findOptions.SetSort(bson.M{"date": -1})

	cur, err := col.Find(ctx, bson.M{"botID": botID, "userID": userID}, findOptions)

	if err == nil || err == mongo.ErrNoDocuments {

		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var vote struct {
				Date int64 `bson:"date"`
			}

			err := cur.Decode(&vote)

			if err != nil {
				return nil, err
			}

			votes = append(votes, vote.Date)
		}
	} else {
		return nil, err
	}

	voteParsed := types.UserVote{
		VoteTime: GetVoteTime(),
	}

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
