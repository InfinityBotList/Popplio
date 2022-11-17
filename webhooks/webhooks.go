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
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
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

	if !webhook.Test && (utils.IsNone(url) || utils.IsNone(token)) {
		// Fetch URL from postgres

		var bot struct {
			Discord    pgtype.Text `db:"webhook"`
			CustomURL  pgtype.Text `db:"custom_webhook"`
			CustomAuth pgtype.Text `db:"web_auth"`
			APIToken   pgtype.Text `db:"token"`
			HMACAuth   pgtype.Bool `db:"hmac"`
		}

		err := pgxscan.Get(state.Context, state.Pool, &bot, "SELECT webhook, custom_webhook, web_auth, token, hmac FROM bots WHERE bot_id = $1", webhook.BotID)

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

		// For each url, make a new sendWebhook
		if !utils.IsNone(bot.CustomURL.String) {
			webhook.URL = bot.CustomURL.String
			err := Send(webhook)
			state.Logger.Error("Custom URL send error", err)
		}

		if !utils.IsNone(bot.Discord.String) {
			webhook.URL = bot.Discord.String
			err := Send(webhook)
			state.Logger.Error("Discord send error", err)
		}
	}

	if utils.IsNone(url) {
		state.Logger.Error("Refusing to continue as no webhook")
		return nil
	}

	if isDiscordIntegration && !isDiscord(url) {
		return errors.New("webhook is not a discord webhook")
	}

	if isDiscordIntegration {
		parts := strings.Split(url, "/")
		if len(parts) < 7 {
			state.Logger.With(
				zap.String("url", url),
			).Warn("Could not parse webhook URL")
			return errors.New("invalid discord webhook URL. Could not parse")
		}

		webhookId := parts[5]
		webhookToken := parts[6]
		userObj, err := utils.GetDiscordUser(webhook.UserID)

		if err != nil {
			userObj = &types.DiscordUser{
				ID:            "510065483693817867",
				Username:      "Toxic Dev (test webhook)",
				Avatar:        "https://cdn.discordapp.com/avatars/510065483693817867/a_96c9cea3c656deac48f1d8fdfdae5007.gif?size=1024",
				Discriminator: "0000",
			}
		}

		state.Logger.With(
			zap.String("user", webhook.UserID),
			zap.String("webhookId", webhookId),
		).Info("Got here in parsing webhook for discord")

		botObj, err := utils.GetDiscordUser(webhook.BotID)
		if err != nil {
			state.Logger.With(
				zap.String("bot", webhook.BotID),
				zap.Error(err),
			).Warn("Could not get bot user")
			return err
		}
		userWithDisc := userObj.Username + "#" + userObj.Discriminator // Create the user object

		var embeds []*discordgo.MessageEmbed = []*discordgo.MessageEmbed{
			{
				Title: "Congrats! " + botObj.Username + " got a new vote!!!",
				Description: "**" + userWithDisc + "** just voted for **" + botObj.Username + "**!\n\n" +
					"**" + botObj.Username + "** now has **" + strconv.Itoa(webhook.Votes) + "** votes!",
				Color: 0x00ff00,
				URL:   "https://botlist.site/bots/" + webhook.BotID,
			},
		}

		_, err = state.Discord.WebhookExecute(webhookId, webhookToken, true, &discordgo.WebhookParams{
			Embeds:    embeds,
			Username:  userObj.Username,
			AvatarURL: userObj.Avatar,
		})

		if err != nil {
			state.Logger.With(
				zap.String("webhookId", webhookId),
				zap.Error(err),
			).Warn("Could not execute webhook")
			return err
		}
	} else {
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
	}

	return nil
}
