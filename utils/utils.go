package utils

import (
	"math/rand"
	"strings"
	"time"
	"unsafe"

	"popplio/types"
)

func isNone(s *string) bool {
	if *s == "None" || *s == "none" || *s == "" || *s == "null" {
		return true
	}
	return false
}

func ParseBot(bot *types.Bot) *types.Bot {
	bot.Tags = strings.Split(strings.ReplaceAll(bot.TagsRaw, " ", ""), ",")

	if isNone(bot.Website) {
		bot.Website = nil
	}

	if isNone(bot.Donate) {
		bot.Donate = nil
	}

	if isNone(bot.Github) {
		bot.Github = nil
	}

	return bot
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
