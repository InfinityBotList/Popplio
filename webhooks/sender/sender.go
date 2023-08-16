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
	"popplio/webhooks/events"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// The Secret
type Secret struct {
	UseInsecure bool // whether to use insecure mode (no encryption), used by legacy webhooks
	Raw         string
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

// An abstraction over an entity whether that be a bot (or teams if we add that in the future, which is very likely)
type WebhookEntity struct {
	// the id of the webhook's target
	EntityID string

	// the entity type
	EntityType string

	// the name of the webhook's target
	EntityName string

	// whether or not the secret is 'insecure' or not
	InsecureSecret bool
}

func (e WebhookEntity) Validate() bool {
	return e.EntityID != "" && e.EntityType != "" && e.EntityName != ""
}

func (st *WebhookSendState) cancelSend(saveState string) {
	state.Logger.Warnf("Cancelling webhook send for %s", st.LogID)

	_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1, tries = tries + 1 WHERE id = $2", saveState, st.LogID)

	if err != nil {
		state.Logger.Errorf("Failed to update webhook state for %s: %s", st.LogID, err.Error())
	}
}

// Creates a webhook response, retrying if needed
func Send(d *WebhookSendState) error {
	if !d.Entity.Validate() {
		panic("invalid webhook entity")
	}

	if d.wdata == nil {
		var url, secret string
		var broken bool

		err := state.Pool.QueryRow(state.Context, "SELECT url, secret, broken FROM webhooks WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType).Scan(&url, &secret, &broken)

		if errors.Is(err, pgx.ErrNoRows) {
			state.Logger.Error("webhook not found for " + d.Entity.EntityID)
			return errors.New("webhook not found")
		}

		if err != nil {
			return err
		}

		if broken {
			state.Logger.Error("webhook is broken for " + d.Entity.EntityID)
			return errors.New("webhook has been flagged for not working correctly")
		}

		d.wdata = &webhookData{
			url: url,
			sign: Secret{
				Raw:         secret,
				UseInsecure: d.Entity.InsecureSecret,
			},
		}
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
			state.Logger.Error(err)
			return err
		}

		if ok {
			return nil
		}

		d.Data, err = json.Marshal(d.Event)

		if err != nil {
			state.Logger.Error(err)
			return errors.New("failed to marshal webhook payload")
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
							Raw:         crypto.RandString(128),
							UseInsecure: d.Entity.InsecureSecret,
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

	state.Logger.With(
		"entityID", d.Entity.EntityID,
		"userId", d.UserID,
	)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	var req *http.Request
	var err error

	if d.wdata.sign.UseInsecure {
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
	req.Header.Set("User-Agent", "Popplio/v7.0.0 (https://infinitybots.gg)")

	resp, err := client.Do(req)

	if err != nil {
		state.Logger.Error(err)

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
		state.Logger.Error(err)
	}

	switch {
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		// Remove from DB
		d.cancelSend("WEBHOOK_BROKEN_404_410")

		_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType)

		if err != nil {
			state.Logger.Error(err)
			return err
		}

		err = notifications.PushNotification(d.UserID, types.Alert{
			Type:    types.AlertTypeWarning,
			Message: "This bot seems to not have a working rewards system.",
			Title:   "Whoa!",
		})

		if err != nil {
			state.Logger.Error(err)
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
				state.Logger.Error(err)
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
			state.Logger.Error(err)
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
				state.Logger.Error(err)
			}

			// Set webhook to broken
			_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType)

			if err != nil {
				state.Logger.Error(err)
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
			state.Logger.Error(err)
		}
	}

	return nil
}

func sendDiscord(userId, url string, entity WebhookEntity, params *discordgo.WebhookParams) (validUrl bool, err error) {
	state.Logger.Info("discord webhook send: ")

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
				state.Logger.Error(err)
				return true, err
			}
		}
	}

	state.Logger.With(
		"url", url,
		"statusCode", resp.StatusCode,
	).Info("sent discord webhook")

	err = notifications.PushNotification(userId, types.Alert{
		Type:    types.AlertTypeSuccess,
		Message: "Successfully notified " + entity.EntityName + " of this action.",
		Title:   "Webhook Send Successful!",
	})

	if err != nil {
		state.Logger.Error(err)
	}

	return true, nil
}

// The data required to create a pull
type WebhookPullPending struct {
	// the entity type
	EntityType string

	// get an entity
	GetEntity func(id string) (WebhookEntity, error)

	// If a entity may not support pulls, implement this function to determine if it does
	// If this function is not implemented, it will be assumed that the entity supports pulls
	SupportsPulls func(id string) (bool, error)
}

// Pulls all pending webhooks from the database and sends them
//
// Do not call this directly/normally, this is handled automatically in 'core'
func PullPending(p WebhookPullPending) {
	if p.SupportsPulls != nil {
		// Check if the entity supports pulls
		supports, err := p.SupportsPulls("")

		if err != nil {
			state.Logger.Error(err)
			return
		}

		if !supports {
			return
		}
	}

	// Fetch every pending bot webhook from webhook_logs
	rows, err := state.Pool.Query(state.Context, "SELECT id, target_id, user_id, data FROM webhook_logs WHERE state = $1 AND target_type = $2 AND bad_intent = false", "PENDING", p.EntityType)

	if err != nil {
		state.Logger.Error(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			id       string
			targetId string
			userId   string
			data     []byte
		)

		err := rows.Scan(&id, &targetId, &userId, &data)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		entity, err := p.GetEntity(targetId)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		entity.EntityType = p.EntityType

		// Send webhook
		err = Send(&WebhookSendState{
			Data:   data,
			LogID:  id,
			UserID: userId,
			Entity: entity,
		})

		if err != nil {
			state.Logger.Error(err)
		}
	}
}
