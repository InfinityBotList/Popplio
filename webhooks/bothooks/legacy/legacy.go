package legacy

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"time"

	"popplio/state"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type WebhookPostLegacy struct {
	BotID  string `json:"bot_id" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
	Test   bool   `json:"test"`
	Votes  int    `json:"votes" validate:"required"`
}

type WebhookStateLegacy struct {
	HTTP      bool `json:"http"`
	SecretSet bool `json:"webhook_secret_set"`
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
	// Fetch URL from postgres
	var webhookURL pgtype.Text
	var webhookSecret pgtype.Text
	var apiToken string

	err := state.Pool.QueryRow(state.Context, "SELECT webhook, web_auth, api_token FROM bots WHERE bot_id = $1", webhook.BotID).Scan(&webhookURL, &webhookSecret, &apiToken)

	if err != nil {
		state.Logger.Error("Failed to fetch webhook: ", err.Error())
		return err
	}

	var token string
	if webhookSecret.Valid && !utils.IsNone(webhookSecret.String) {
		token = webhookSecret.String
	} else {
		token = apiToken
	}

	isDiscordIntegration, _ := isDiscordAPIURL(webhookURL.String)

	if isDiscordIntegration {
		return errors.New("only supported on v2")
	}

	if utils.IsNone(webhookURL.String) {
		return errors.New("no webhook set")
	}

	dUser, err := dovewing.GetDiscordUser(state.Context, webhook.UserID)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	// Create response body
	body := WebhookDataLegacy{
		Votes:        webhook.Votes,
		UserID:       webhook.UserID,
		UserObj:      dUser,
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

	var finalToken string = token

	// Create request
	responseBody := bytes.NewBuffer(data)
	req, err := http.NewRequestWithContext(state.Context, "POST", webhookURL.String, responseBody)

	if err != nil {
		state.Logger.Error("Failed to create request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "popplio/legacyhandler")
	req.Header.Set("Authorization", finalToken)

	// Send request
	client := &http.Client{Timeout: time.Second * 5}
	_, err = client.Do(req)

	if err != nil {
		state.Logger.Error("Failed to send request")
		return err
	}

	return nil
}

func Setup() {
	docs.AddWebhook(&docs.WebhookDoc{
		Name:    "Legacy",
		Summary: "Legacy Webhooks",
		Tags: []string{
			"Webhooks",
		},
		Description: `(older) v1 webhooks. Only supports Votes`,
		Format:      WebhookDataLegacy{},
		FormatName:  "WebhookLegacyResponse",
	})
}
