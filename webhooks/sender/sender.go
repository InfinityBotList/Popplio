package sender

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	rand2 "math/rand"
	"net/http"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// The Secret
type Secret struct {
	SimpleAuth bool // whether to use simple auth mode (no encryption) or not
	Raw        string
}

func (s Secret) Sign(data []byte) string {
	h := hmac.New(sha512.New, []byte(s.Raw))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

type webhookData struct {
	sign Secret
	url  string
}

// Internal structs
type WebhookSendState struct {
	// webhook event (used for discord webhooks)
	Event *events.WebhookResponse

	// the data to send
	Data []byte

	// is it a bad intent: intentionally bad auth to trigger 401 check
	BadIntent bool

	// Automatically set fields
	LogID string

	// user id that triggered the webhook
	UserID string

	// The entity itself
	Entity WebhookEntity

	// low-level data
	wdata *webhookData
}

// An abstraction over an entity whether that be a bot/team/server
type WebhookEntity struct {
	// the id of the webhook's target
	EntityID string

	// the entity type
	EntityType string

	// the name of the webhook's target
	EntityName string

	// Override whether or not the authentication is 'simple' (no auth header) or not
	//
	// TODO: Hack until legacy webhooks is truly removed
	SimpleAuth *bool
}

func (e WebhookEntity) Validate() bool {
	return e.EntityID != "" && e.EntityType != "" && e.EntityName != ""
}

func (st *WebhookSendState) cancelSend(saveState string) {
	state.Logger.Warn("Cancelling webhook send", zap.String("logID", st.LogID), zap.String("userID", st.UserID), zap.String("entityID", st.Entity.EntityID), zap.Bool("badIntent", st.BadIntent))

	_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1, tries = tries + 1 WHERE id = $2", saveState, st.LogID)

	if err != nil {
		state.Logger.Error("Failed to update webhook logs with new status", zap.Error(err), zap.String("logID", st.LogID), zap.String("userID", st.UserID), zap.String("entityID", st.Entity.EntityID), zap.Bool("badIntent", st.BadIntent))
	}
}

// Creates a webhook response, retrying if needed
func Send(d *WebhookSendState) error {
	if !d.Entity.Validate() {
		panic("invalid webhook entity")
	}

	if d.wdata == nil {
		var url, secret string
		var broken, simpleAuth bool

		err := state.Pool.QueryRow(state.Context, "SELECT url, secret, broken, simple_auth FROM webhooks WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType).Scan(&url, &secret, &broken, &simpleAuth)

		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("webhook not found for %s", d.Entity.EntityID)
		}

		if err != nil {
			return err
		}

		if broken {
			state.Logger.Error("webhook is broken for " + d.Entity.EntityID)
			return fmt.Errorf("webhook has been flagged as broken for %s", d.Entity.EntityID)
		}

		d.wdata = &webhookData{
			url: url,
			sign: Secret{
				Raw:        secret,
				SimpleAuth: simpleAuth,
			},
		}
	}

	// Override simpleauth flag if requested/set
	if d.Entity.SimpleAuth != nil {
		d.wdata.sign.SimpleAuth = *d.Entity.SimpleAuth
	}

	// Handle webhook event
	if d.Event != nil {
		params := d.Event.Data.CreateHookParams(d.Event.Creator, d.Event.Targets)

		ok, err := sendDiscord(
			d.Event.Creator.ID,
			d.wdata.url,
			d.Entity,
			params,
		)

		if err != nil {
			state.Logger.Error("failed to send discord webhook", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
			return err
		}

		if ok {
			return nil
		}

		d.Data, err = json.Marshal(d.Event)

		if err != nil {
			state.Logger.Error("failed to marshal webhook payload", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}
	}

	// Randomly send a bad webhook with invalid auth
	if !d.BadIntent {
		if rand2.Float64() < 0.4 {
			go func() {
				badD := &WebhookSendState{
					BadIntent: true,
					wdata: &webhookData{
						url: d.wdata.url,
						sign: Secret{
							Raw:        crypto.RandString(128),
							SimpleAuth: d.wdata.sign.SimpleAuth,
						},
					},
					Data:   d.Data,
					UserID: d.UserID,
					Entity: d.Entity,
				}

				// Retry with bad intent
				Send(badD)
			}()
		}
	}

	if d.LogID == "" {
		// Add to webhook logs for automatic retry
		var logID string
		err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (target_id, target_type, user_id, url, data, bad_intent) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", d.Entity.EntityID, d.Entity.EntityType, d.UserID, d.wdata.url, d.Data, d.BadIntent).Scan(&logID)

		if err != nil {
			return err
		}

		d.LogID = logID
	}

	state.Logger.Info("Sending webhook", zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	var req *http.Request
	var err error

	if d.wdata.sign.SimpleAuth {
		req, err = http.NewRequestWithContext(state.Context, "POST", d.wdata.url, bytes.NewReader(d.Data))

		if err != nil {
			return err
		}

		req.Header.Set("Authorization", d.wdata.sign.Raw)
		req.Header.Set("X-Webhook-Protocol", "legacy-insecure")
	} else {
		// Generate HMAC token using nonce and signed header for further randomization
		nonce := crypto.RandString(16)

		keyHash := sha256.New()
		keyHash.Write([]byte(d.wdata.sign.Raw + nonce))

		// Encrypt request body with hashed
		c, err := aes.NewCipher(keyHash.Sum(nil))

		if err != nil {
			return err
		}

		gcm, err := cipher.NewGCM(c)

		if err != nil {
			return err
		}

		aesNonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, aesNonce); err != nil {
			return err
		}

		postData := []byte(hex.EncodeToString(gcm.Seal(aesNonce, aesNonce, d.Data, nil)))

		// HMAC with encrypted request body
		tok1 := d.wdata.sign.Sign(postData)

		finalToken := Secret{Raw: nonce}.Sign([]byte(tok1))

		req, err = http.NewRequestWithContext(state.Context, "POST", d.wdata.url, bytes.NewReader(postData))

		if err != nil {
			return err
		}

		req.Header.Set("X-Webhook-Signature", finalToken)
		req.Header.Set("X-Webhook-Protocol", "splashtail")
		req.Header.Set("X-Webhook-Nonce", nonce)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "Popplio/v8.0.0 (https://infinitybots.gg)")

	resp, err := client.Do(req)

	if err != nil {
		state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))

		d.cancelSend("REQUEST_SEND_FAILURE")
		return err
	}

	// Only read a maximum of 1kb, with timeout of 65 seconds
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if err != nil {
		body = []byte("Failed to read body: " + err.Error())
	}

	// Set response to body
	_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET response = $1, status_code = $2 WHERE id = $3", body, resp.StatusCode, d.LogID)

	if err != nil {
		state.Logger.Error("Failed to update webhook logs with response", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
	}

	switch {
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		// Remove from DB
		d.cancelSend("WEBHOOK_BROKEN_404_410")

		_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType)

		if err != nil {
			state.Logger.Error("Failed to update webhook logs with response", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
			return fmt.Errorf("webhook failed to validate auth and failed to remove webhook from db: %w", err)
		}

		err = notifications.PushNotification(d.UserID, types.Alert{
			Type:    types.AlertTypeWarning,
			Message: "This bot seems to not have a working rewards system.",
			Title:   "Whoa!",
		})

		if err != nil {
			state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
		}

		return errors.New("webhook returned not found thus removing it from the database")

	case resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 418:
		if d.BadIntent {
			// webhook auth is invalid as intended,
			d.cancelSend("SUCCESS")

			return nil
		} else {
			// webhook auth is invalid, return error
			d.cancelSend("WEBHOOK_AUTH_INVALID")
			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    types.AlertTypeInfo,
				Message: "Webhook could not be securely authenticated by the bot at this time. Please try again later.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
			}

			return errors.New("webhook auth error:" + strconv.Itoa(resp.StatusCode))
		}

	case resp.StatusCode > 400:
		d.cancelSend("RESPONSE_" + strconv.Itoa(resp.StatusCode))

		err = notifications.PushNotification(d.UserID, types.Alert{
			Type:    types.AlertTypeError,
			Message: fmt.Sprintf("We were unable to notify this bot: %d", resp.StatusCode),
			Title:   "Webhook Auth Error",
		})

		if err != nil {
			state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
		}

		return errors.New("webhook returned error: " + strconv.Itoa(resp.StatusCode))

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if d.BadIntent {
			d.cancelSend("WEBHOOK_BROKEN_BAD_AUTHCODE")

			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    types.AlertTypeError,
				Message: "This webhook does not properly handle authentication at this time.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
			}

			// Set webhook to broken
			_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType)

			if err != nil {
				return errors.New("webhook failed to validate auth and failed to remove webhook from db")
			}

			return errors.New("webhook failed to validate auth thus removing it from the database")
		}

		d.cancelSend("SUCCESS")

		err = notifications.PushNotification(d.UserID, types.Alert{
			Type:    types.AlertTypeSuccess,
			Message: "Successfully notified " + d.Entity.EntityName + " of this action.",
			Title:   "Webhook Send Successful!",
		})

		if err != nil {
			state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
		}
	}

	return nil
}

func sendDiscord(userId, url string, entity WebhookEntity, params *discordgo.WebhookParams) (validUrl bool, err error) {
	validPrefixes := []string{
		"https://discordapp.com/",
		"https://discord.com/",
		"https://canary.discord.com/",
		"https://ptb.discord.com/",
	}

	var flag bool
	var prefix string
	for _, p := range validPrefixes {
		if strings.HasPrefix(url, p) {
			flag = true
			prefix = p
			break
		}
	}

	if !flag {
		return false, nil
	}

	// Remove out prefix
	url = state.Config.Meta.PopplioProxy + "/" + strings.TrimPrefix(url, prefix)

	if !strings.Contains(url, "/webhooks/") {
		return true, errors.New("invalid discord webhook url")
	}

	payload, err := json.Marshal(params)

	if err != nil {
		return true, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))

	if err != nil {
		return true, err
	}

	for _, code := range []int{404, 401, 403, 410} {
		if resp.StatusCode == code {
			// This webhook is broken
			_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2", entity.EntityID, entity.EntityType)

			if err != nil {
				state.Logger.Error("Failed to update webhook logs with response", zap.Error(err), zap.String("userID", userId), zap.String("entityID", entity.EntityID), zap.String("entityType", entity.EntityType), zap.Int("status", resp.StatusCode))
				return true, err
			}
		}
	}

	state.Logger.Info("discord webhook send", zap.Int("status", resp.StatusCode), zap.String("url", url), zap.String("entityID", entity.EntityID), zap.String("userID", userId))

	err = notifications.PushNotification(userId, types.Alert{
		Type:    types.AlertTypeSuccess,
		Message: "Successfully notified " + entity.EntityName + " of this action.",
		Title:   "Webhook Send Successful!",
	})

	if err != nil {
		state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("userID", userId), zap.String("entityID", entity.EntityID))
	}

	return true, nil
}
