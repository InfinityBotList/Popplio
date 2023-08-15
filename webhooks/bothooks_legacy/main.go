package bothooks_legacy

import (
	"strings"
	"time"

	"popplio/state"
	"popplio/webhooks/sender"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	jsoniter "github.com/json-iterator/go"
)

const EntityType = "bot"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type WebhookPostLegacy struct {
	BotID  string `json:"bot_id" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
	Test   bool   `json:"test"`
	Votes  int    `json:"votes" validate:"required"`
}

type WebhookDataLegacy struct {
	Votes        int                     `json:"votes"`
	UserID       string                  `json:"user"`
	UserObj      *dovetypes.PlatformUser `json:"userObj"`
	BotObj       *dovetypes.PlatformUser `json:"botObj"`
	BotID        string                  `json:"bot"`
	UserIDLegacy string                  `json:"userID"`
	BotIDLegacy  string                  `json:"botID"`
	Test         bool                    `json:"test"`
	Time         int64                   `json:"time"`
}

func isDiscordAPIURL(url string) (bool, string) {
	validPrefixes := []string{
		"https://discordapp.com/",
		"https://discord.com/",
		"https://canary.discord.com/",
		"https://ptb.discord.com/",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true, prefix
		}
	}

	return false, ""
}

// Sends a webhook using the legacy v1 format
func SendLegacy(webhook WebhookPostLegacy) error {
	dUser, err := dovewing.GetUser(state.Context, webhook.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	dBot, err := dovewing.GetUser(state.Context, webhook.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	// Create response body
	body := WebhookDataLegacy{
		Votes:        webhook.Votes,
		UserID:       webhook.UserID,
		UserObj:      dUser,
		BotObj:       dBot,
		BotID:        webhook.BotID,
		UserIDLegacy: webhook.UserID,
		BotIDLegacy:  webhook.BotID,
		Test:         webhook.Test,
		Time:         time.Now().Unix(),
	}

	data, err := json.Marshal(body)

	if err != nil {
		state.Logger.Error("Failed to encode data")
		return err
	}

	// Send webhook
	entity := sender.WebhookEntity{
		EntityID:       webhook.BotID,
		EntityType:     EntityType,
		EntityName:     dBot.Username,
		InsecureSecret: true,
	}

	return sender.Send(&sender.WebhookSendState{
		Data:   data,
		UserID: webhook.UserID,
		Entity: entity,
	})
}
