package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/infinitybotlist/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func IsDiscordURL(url string) bool {
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

// Sends a webhook using the legacy v1 format
func SendLegacy(webhook types.WebhookPostLegacy) error {
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

	if url == "httpUser" {
		return errors.New("httpUser")
	}

	isDiscordIntegration := IsDiscordURL(url)

	if isDiscordIntegration {
		return errors.New("webhook is not a discord webhook")
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

	var tries = 0

	for tries < 3 {
		// Create response body
		body := types.WebhookDataLegacy{
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
		req.Header.Set("User-Agent", "popplio/v6.0")
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
