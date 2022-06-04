package utils

import (
	"strings"

	"popplio/types"
)

func ParseBot(bot *types.Bot) *types.Bot {
	bot.Tags = strings.Split(strings.ReplaceAll(bot.TagsRaw, " ", ""), ",")

	if *bot.Website == "None" || *bot.Website == "none" || *bot.Website == "" {
		bot.Website = nil
	}

	if *bot.Donate == "None" || *bot.Donate == "none" || *bot.Donate == "" {
		bot.Donate = nil
	}

	if *bot.Github == "None" || *bot.Github == "none" || *bot.Github == "" {
		bot.Github = nil
	}

	return bot
}
