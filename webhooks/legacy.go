package webhooks

import (
	"bytes"
	"errors"
	"net/http"
	"time"

	"popplio/state"
	"popplio/utils"

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

	// Only present on test webhook API or during sends internally
	URL string `json:"url" validate:"required"`

	// Only present on test webhook API
	Token string `json:"token" validate:"required"`

	// Only present on test webhook API
	HMACAuth bool `json:"hmac_auth"`
}

type WebhookStateLegacy struct {
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

// Sends a webhook using the legacy v1 format
func SendLegacy(webhook WebhookPostLegacy) error {
	url, token := webhook.URL, webhook.Token

	if utils.IsNone(url) || utils.IsNone(token) {
		// Fetch URL from postgres

		var webhookURL pgtype.Text
		var webhookSecret pgtype.Text
		var apiToken string

		err := state.Pool.QueryRow(state.Context, "SELECT webhook, web_auth, api_token FROM bots WHERE bot_id = $1", webhook.BotID).Scan(&webhookURL, &webhookSecret, &apiToken)

		if err != nil {
			state.Logger.Error("Failed to fetch webhook: ", err.Error())
			return err
		}

		if webhookSecret.Valid && !utils.IsNone(webhookSecret.String) {
			token = webhookSecret.String
		} else {
			token = apiToken
		}

		webhook.HMACAuth = false
		webhook.Token = token

		url = webhookURL.String
	}

	isDiscordIntegration, _ := isDiscordAPIURL(url)

	if isDiscordIntegration {
		return errors.New("only supported on v2")
	}

	state.Logger.Info("Using hmac: ", webhook.HMACAuth)

	if utils.IsNone(url) {
		return errors.New("no webhook set, vote rewards may not work")
	}

	var dUser, err = dovewing.GetDiscordUser(state.Context, webhook.UserID)

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

	var finalToken string = webhook.Token

	// Create request
	responseBody := bytes.NewBuffer(data)
	req, err := http.NewRequestWithContext(state.Context, "POST", url, responseBody)

	if err != nil {
		state.Logger.Error("Failed to create request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "popplio/legacyhandler-1")
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
