package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
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

		var bot struct {
			WebhookURL pgtype.Text `db:"webhook"`
			CustomAuth pgtype.Text `db:"web_auth"`
			APIToken   pgtype.Text `db:"token"`
			HMACAuth   pgtype.Bool `db:"hmac"`
		}

		err := pgxscan.Get(state.Context, state.Pool, &bot, "SELECT webhook, web_auth, token, hmac FROM bots WHERE bot_id = $1", webhook.BotID)

		if err != nil {
			state.Logger.Error("Failed to fetch webhook: ", err.Error())
			return err
		}

		// Check custom auth viability
		if !bot.CustomAuth.Valid || utils.IsNone(bot.CustomAuth.String) {
			if bot.APIToken.String != "" {
				token = bot.APIToken.String
			} else {
				// We set the token to the a random string in DB in this case
				token = utils.RandString(256)

				_, err := state.Pool.Exec(state.Context, "UPDATE bots SET web_auth = $1 WHERE bot_id = $2", token, webhook.BotID)

				if err != pgx.ErrNoRows && err != nil {
					state.Logger.Error("Failed to update webhook: ", err.Error())
					return err
				}
			}

			bot.CustomAuth = pgtype.Text{String: token, Valid: true}
		}

		webhook.HMACAuth = bot.HMACAuth.Bool
		webhook.Token = bot.CustomAuth.String

		state.Logger.Info("Using hmac: ", webhook.HMACAuth)

		url = bot.WebhookURL.String
	}

	if utils.IsNone(url) {
		return errors.New("refusing to continue as no webhook")
	}

	if isDiscordIntegration || isDiscord(url) {
		state.Logger.Info("Sending discord webhook has been disabled:", url)
		return nil
	}

	tries := 0

	for tries < 3 {
		if webhook.Test {
			webhook.UserID = "510065483693817867"
		}

		var dUser, err = utils.GetDiscordUser(webhook.UserID)

		if err != nil {
			state.Logger.Error(err)
		}

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

		if webhook.HMACAuth {
			// Generate HMAC token using token and request body
			h := hmac.New(sha512.New, []byte(token))
			h.Write(data)
			token = hex.EncodeToString(h.Sum(nil))
		}

		// Create request
		responseBody := bytes.NewBuffer(data)
		req, err := http.NewRequest("POST", url, responseBody)

		if err != nil {
			state.Logger.Error("Failed to create request")
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Popplio/v5.0")
		req.Header.Set("Authorization", token)

		// Send request
		client := &http.Client{Timeout: time.Second * 5}
		resp, err := client.Do(req)

		if err != nil {
			state.Logger.Error("Failed to send request")
			return err
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
