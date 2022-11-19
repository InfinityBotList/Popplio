package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func isDiscord(url string) bool {
	validPrefixes := []string{
		"https://discordapp.com/api/webhooks/",
		"https://discord.com/api/webhooks/",
		"https://canary.discord.com/api/webhooks/",
		"https://ptb.discord.com/api/webhooks/",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
}

// Sends a webhook
func Send(webhook types.WebhookPost) error {
	url, token := webhook.URL, webhook.Token

	isDiscordIntegration := isDiscord(url)

	if isDiscordIntegration {
		return errors.New("webhook is not a discord webhook")
	}

	if !webhook.Test && (utils.IsNone(url) || utils.IsNone(token)) {
		// Fetch URL from postgres

		var webhookURL pgtype.Text
		var webhookSecret pgtype.Text
		var apiToken string
		var hmacAuth bool

		err := state.Pool.QueryRow(state.Context, "SELECT webhook, web_auth, token, hmac FROM bots WHERE bot_id = $1", webhook.BotID).Scan(&webhookURL, &webhookSecret, &apiToken, &hmacAuth)

		if err != nil {
			state.Logger.Error("Failed to fetch webhook: ", err.Error())
			return err
		}

		if webhookSecret.Valid && !utils.IsNone(webhookSecret.String) {
			token = webhookSecret.String
		} else {
			token = apiToken
		}

		webhook.HMACAuth = hmacAuth
		webhook.Token = token

		url = webhookURL.String
	}

	state.Logger.Info("Using hmac: ", webhook.HMACAuth)

	if utils.IsNone(url) {
		return errors.New("refusing to continue as no webhook")
	}

	if isDiscordIntegration || isDiscord(url) {
		state.Logger.Info("Sending discord webhook has been disabled:", url)
		return nil
	}

	if webhook.Test {
		webhook.UserID = "510065483693817867"
	}

	var dUser, err = utils.GetDiscordUser(webhook.UserID)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	var tries = 0

	for tries < 3 {
		// Create response body
		body := types.WebhookData{
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
		if webhook.HMACAuth {
			// Generate HMAC token using token and request body
			h := hmac.New(sha512.New, []byte(webhook.Token))
			h.Write(data)
			finalToken = hex.EncodeToString(h.Sum(nil))
		}

		// Create request
		responseBody := bytes.NewBuffer(data)
		req, err := http.NewRequest("POST", url, responseBody)

		if err != nil {
			state.Logger.Error("Failed to create request")
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Popplio/v6.0")
		req.Header.Set("Authorization", finalToken)

		// Send request
		client := &http.Client{Timeout: time.Second * 5}
		resp, err := client.Do(req)

		if err != nil {
			state.Logger.Error("Failed to send request")
			return err
		}

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			state.Logger.Error("Failed to send request: invalid token")

			// get response body
			body, err := io.ReadAll(resp.Body)

			if err != nil {
				state.Logger.Error("Failed to read response body")
			}

			state.Logger.Info("Response body: ", string(body))

			return errors.New("webhook is broken")
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			state.Logger.Info("Retrying webhook again. Got status code of ", resp.StatusCode)
			tries++
			continue
		}

		break
	}

	return nil
}
