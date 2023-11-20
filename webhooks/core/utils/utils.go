package utils

import (
	"errors"
	"strings"
)

var ErrNotActuallyWebhook = errors.New("webhook url has discord prefix but is not a webhook")

func GetDiscordWebhookInfo(url string) (prefix string, err error) {
	validPrefixes := []string{
		"https://discordapp.com/",
		"https://discord.com/",
		"https://canary.discord.com/",
		"https://ptb.discord.com/",
	}

	for _, p := range validPrefixes {
		if strings.HasPrefix(url, p) {
			if !strings.Contains(url, "/webhooks/") {
				return p, ErrNotActuallyWebhook
			}
			return p, nil
		}
	}

	return "", nil
}
