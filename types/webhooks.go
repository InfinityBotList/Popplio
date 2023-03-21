package types

import (
	"github.com/infinitybotlist/dovewing"
)

type WebhookPostLegacy struct {
	BotID  string `json:"bot_id" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
	Test   bool   `json:"test"`
	Votes  int    `json:"votes" validate:"required"`

	// Only present on test webhook API or during sends internally
	URL string `json:"url" validate:"required"`

	// Only present on test webhook API
	Token string `json:"token" validate:"required"`

	// Only present on test webhook API
	HMACAuth bool `json:"hmac_auth"`
}

type WebhookState struct {
	HTTP       bool `json:"http"`
	WebhooksV2 bool `json:"webhooks_v2"`
	SecretSet  bool `json:"webhook_secret_set"`
}

type WebhookDataLegacy struct {
	Votes        int                   `json:"votes"`
	UserID       string                `json:"user"`
	UserObj      *dovewing.DiscordUser `json:"userObj"`
	BotID        string                `json:"bot"`
	UserIDLegacy string                `json:"userID"`
	BotIDLegacy  string                `json:"botID"`
	Test         bool                  `json:"test"`
	Time         int64                 `json:"time"`
}
