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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	rand2 "math/rand"
	"net/http"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/crypto"
)

// The Secret
type Secret struct {
	Raw string
}

func (s Secret) Sign(data []byte) string {
	h := hmac.New(sha512.New, []byte(s.Raw))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Internal structs
type WebhookSendState struct {
	// the url to post to
	Url string

	// the data to send
	Data []byte

	// the hmac512 signed header to send
	Sign Secret

	// is it a bad intent: intentionally bad auth to trigger 401 check
	BadIntent bool

	// Automatically set fields
	LogID string

	// user id that triggered the webhook
	UserID string

	// The entity itself
	Entity WebhookEntity
}

// An abstraction over an entity whether that be a bot (or teams if we add that in the future, which is very likely)
type WebhookEntity struct {
	// the id of the webhook's target
	EntityID string

	// the entity type
	EntityType string

	// the name of the webhook's target
	EntityName string

	// deletes webhook from entity
	DeleteWebhook func() error
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
	// Randomly send a bad webhook with invalid auth
	if rand2.Float64() < 0.7 {
		go func() {
			badD := &WebhookSendState{
				BadIntent: true,
				Sign: Secret{
					Raw: crypto.RandString(128),
				},
				Url:    d.Url,
				Data:   d.Data,
				UserID: d.UserID,
				Entity: d.Entity,
			}

			// Retry with bad intent
			Send(badD)
		}()
	}

	if d.LogID == "" {
		// Add to webhook logs for automatic retry
		var logID string
		err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (entity_id, entity_type, user_id, url, data, sign, bad_intent) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", d.Entity.EntityID, d.Entity.EntityType, d.UserID, d.Url, d.Data, d.Sign.Raw, d.BadIntent).Scan(&logID)

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
		Timeout: 10 * time.Second,
	}

	// Generate HMAC token using nonce and signed header for further randomization
	nonce := crypto.RandString(16)

	keyHash := sha256.New()
	keyHash.Write([]byte(d.Sign.Raw + nonce))

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
	tok1 := d.Sign.Sign(postData)

	finalToken := Secret{Raw: nonce}.Sign([]byte(tok1))

	req, err := http.NewRequestWithContext(state.Context, "POST", d.Url, bytes.NewReader(postData))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "Popplio/v7.0.0 (https://infinitybots.gg)")
	req.Header.Set("X-Webhook-Signature", finalToken)
	req.Header.Set("X-Webhook-Protocol", "splashtail")
	req.Header.Set("X-Webhook-Nonce", nonce)

	resp, err := client.Do(req)

	if err != nil {
		state.Logger.Error(err)

		d.cancelSend("REQUEST_SEND_FAILURE")
		return err
	}

	switch {
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		// Remove from DB
		d.cancelSend("WEBHOOK_REMOVED_404_410")
		err := d.Entity.DeleteWebhook()

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

	case resp.StatusCode == 401 || resp.StatusCode == 403:
		if d.BadIntent {
			// webhook auth is invalid as intended,
			d.cancelSend("SUCCESS")

			return nil
		} else {
			// webhook auth is invalid, return error
			d.cancelSend("WEBHOOK_AUTH_INVALID")
			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    types.AlertTypeInfo,
				Message: "This webhook does not properly handle authentication at this time.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error(err)
			}

			return errors.New("webhook auth error")
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

		return errors.New("webhook returned error")

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if d.BadIntent {
			d.cancelSend("WEBHOOK_DELETED_BAD_AUTHCODE")

			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    types.AlertTypeError,
				Message: "This webhook does not properly handle authentication at this time.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error(err)
			}

			// Remove webhook, it doesn't validate auth at all
			err := d.Entity.DeleteWebhook()

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

func SendDiscord(userId, entityName, url string, delete func() error, params *discordgo.WebhookParams) (validUrl bool, err error) {
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
			delete()
		}
	}

	state.Logger.With(
		"url", url,
		"statusCode", resp.StatusCode,
	).Info("sent discord webhook")

	err = notifications.PushNotification(userId, types.Alert{
		Type:    types.AlertTypeSuccess,
		Message: "Successfully notified " + entityName + " of this action.",
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

	// delete webhook for specific id
	GetEntity func(id string) (WebhookEntity, error)

	// If a entity may not support pulls, implement this function to determine if it does
	// If this function is not implemented, it will be assumed that the entity supports pulls
	SupportsPulls func(id string) (bool, error)
}

// Pulls all pending webhooks from the database and sends them
//
// Do not call this directly/normally, this is meant for webhook handlers such as “bothooks“
// or a potential “teamhooks“ etc.
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
	rows, err := state.Pool.Query(state.Context, "SELECT id, entity_id, user_id, url, data, sign, bad_intent FROM webhook_logs WHERE state = $1 AND entity_type = $2", "PENDING", p.EntityType)

	if err != nil {
		state.Logger.Error(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			entityId  string
			userId    string
			url       string
			data      []byte
			sign      string
			badIntent bool
		)

		err := rows.Scan(&id, &entityId, &userId, &url, &data, &sign, &badIntent)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		entity, err := p.GetEntity(entityId)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		entity.EntityType = p.EntityType

		// Send webhook
		err = Send(&WebhookSendState{
			Url: url,
			Sign: Secret{
				Raw: sign,
			},
			Data:      data,
			BadIntent: badIntent,
			LogID:     id,
			UserID:    userId,
			Entity:    entity,
		})

		if err != nil {
			state.Logger.Error(err)
		}
	}
}
