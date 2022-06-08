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

func GetDiscordUser(s *discordgo.Session, redisCache *redis.Client, ctx context.Context, id string) (*discordgo.User, error) {
	// Check if in discordgo session first

	var userExpiryTime = 4 * time.Hour

	if s.State != nil {
		guilds := s.State.Guilds

		for _, guild := range guilds {
			member, err := s.State.Member(guild.ID, id)

			if err == nil {
				redisCache.Set(ctx, "uobj:"+id, member, userExpiryTime)
				return member.User, err
			}
		}
	}

	// Check if in redis cache
	userBytes, err := redisCache.Get(ctx, "uobj:"+id).Result()

	if err == nil {
		// Try to unmarshal

		var user discordgo.User

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

	// Store in redis
	redisCache.Set(ctx, "uobj:"+id, user, userExpiryTime)

	return user, nil
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