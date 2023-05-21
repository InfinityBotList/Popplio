package bothooks_legacy

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/state"
	"popplio/utils"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

const EntityType = "BOT_LEGACY"

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

func cancelSend(logID string, saveState string) {
	state.Logger.Warnf("Cancelling webhook send for %s", logID)

	_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1, tries = tries + 1 WHERE id = $2", saveState, logID)

	if err != nil {
		state.Logger.Errorf("Failed to update webhook state for %s: %s", logID, err.Error())
	}
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

	// Save request to webhook logs
	var logID string
	err = state.Pool.QueryRow(
		state.Context,
		"INSERT INTO webhook_logs (entity_id, entity_type, user_id, url, data, sign, bad_intent) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		webhook.BotID,
		EntityType,
		webhook.UserID,
		webhookURL.String,
		data,
		"@apiToken",
		false,
	).Scan(&logID)

	if err != nil {
		return err
	}

	state.Logger.Info("Saved webhook log: ", logID)

	// Create request
	responseBody := bytes.NewBuffer(data)
	req, err := http.NewRequestWithContext(state.Context, "POST", webhookURL.String, responseBody)

	if err != nil {
		state.Logger.Error("Failed to create request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "popplio/legacyhandler")
	req.Header.Set("Authorization", token)

	// Send request
	client := &http.Client{Timeout: time.Second * 5}
	resp, err := client.Do(req)

	if err != nil {
		state.Logger.Error("Failed to send request")
		cancelSend(logID, "REQUEST_SEND_FAILURE")
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		cancelSend(logID, "SUCCESS")
	} else {
		cancelSend(logID, "RESPONSE_"+strconv.Itoa(resp.StatusCode))
	}

	return nil
}
