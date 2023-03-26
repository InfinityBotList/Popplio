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
	"io"
	"net/http"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"strconv"
	"time"

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
	// tries already attempted
	Tries int

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
	EntityType WebhookEntityType

	// the name of the webhook's target
	EntityName string

	// deletes webhook from entity
	DeleteWebhook func() error
}

// Number of times to retry sending a webhook
const MaxWebhookTries = 5

// The state of the webhook in the database
type WebhookSaveState int

const (
	WebhookSaveStatePending WebhookSaveState = iota
	WebhookSaveStateSuccess
	WebhookSaveStateFailed
	WebhookSaveStateRemoved
)

type WebhookEntityType int

const (
	WebhookEntityTypeBot WebhookEntityType = iota
)

func (st *WebhookSendState) cancelSend(saveState WebhookSaveState) {
	st.Tries = MaxWebhookTries

	state.Logger.Warnf("Cancelling webhook send for %s", st.LogID)

	_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1 WHERE id = $2", saveState, st.LogID)

	if err != nil {
		state.Logger.Errorf("Failed to update webhook state for %s: %s", st.LogID, err.Error())
	}
}

// Creates a custom webhook response, retrying if needed
func SendCustom(d *WebhookSendState) error {
	d.Tries++

	if d.LogID == "" {
		// Add to webhook logs for automatic retry
		var logID string
		err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (entity_id, entity_type, user_id, url, data, sign, bad_intent) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", d.Entity.EntityID, d.Entity.EntityType, d.UserID, d.Url, d.Data, d.Data, d.BadIntent).Scan(&logID)

		if err != nil {
			return err
		}

		d.LogID = logID
	} else {
		// Update webhook logs
		_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET tries = tries + 1 WHERE id = $1", d.LogID)

		if err != nil {
			return err
		}
	}

	if d.Tries > MaxWebhookTries {
		_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.LogID, WebhookSaveStateFailed)

		if err != nil {
			return err
		}

		return errors.New("too many tries")
	}

	state.Logger.With(
		"entityID", d.Entity.EntityID,
		"userId", d.UserID,
		"tries", d.Tries,
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
		time.Sleep(5 * time.Minute)
		return SendCustom(d)
	}

	switch {
	case resp.StatusCode >= 429:
		// Retry after
		retryAfter := resp.Header.Get("Retry-After")

		if retryAfter == "" {
			time.Sleep(5 * time.Minute)
			return SendCustom(d)
		}

		retryAfterInt, err := strconv.Atoi(retryAfter)

		if err != nil {
			state.Logger.With(
				"retryAfter", retryAfter,
			).Error(err)
			time.Sleep(5 * time.Minute)
			return SendCustom(d)
		}

		time.Sleep(time.Duration(retryAfterInt+d.Tries^2+15) * time.Second)
		return SendCustom(d)

	case resp.StatusCode == 404 || resp.StatusCode == 410:
		// Remove from DB
		d.cancelSend(WebhookSaveStateFailed)
		err := d.Entity.DeleteWebhook()

		if err != nil {
			state.Logger.Error(err)
			return err
		}

		// Remove from webhook logs
		_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.LogID, WebhookSaveStateRemoved)

		if err != nil {
			state.Logger.Error(err)
			return err
		}

		return errors.New("webhook returned not found thus removing it from the database")

	case resp.StatusCode == 401 || resp.StatusCode == 403:
		if d.BadIntent {
			// webhook auth is invalid as intended,
			d.cancelSend(WebhookSaveStateSuccess)

			return nil
		} else {
			// webhook auth is invalid, return error
			d.cancelSend(WebhookSaveStateFailed)
			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    "info",
				Message: "This webhook does not properly handle authentication at this time.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error(err)
			}

			return errors.New("webhook auth error")
		}

	case resp.StatusCode > 400:
		return errors.New("webhook returned error")

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if d.BadIntent {
			d.cancelSend(WebhookSaveStateRemoved)

			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    "info",
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

			// Remove from webhook logs
			_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.LogID, WebhookSaveStateRemoved)

			if err != nil {
				state.Logger.Error(err)
				return errors.New("webhook failed to validate auth and failed to remove webhook from logdb")
			}

			return errors.New("webhook failed to validate auth thus removing it from the database")
		} else {
			d.cancelSend(WebhookSaveStateSuccess)
		}
	case resp.StatusCode >= 500:
		// Give 15 minutes to recover
		time.Sleep(15 * time.Minute)
		return SendCustom(d)
	}

	_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.LogID, WebhookSaveStateSuccess)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	err = notifications.PushNotification(d.UserID, types.Alert{
		Type:    "success",
		Message: "Successfully notified " + d.Entity.EntityName + " of this action.",
		Title:   "Webhook Send Successful!",
	})

	if err != nil {
		state.Logger.Error(err)
	}

	return nil
}

// The data required to create a pull
type WebhookPullPending struct {
	// the entity type
	EntityType WebhookEntityType

	// delete webhook for specific id
	GetEntity func(id string) (WebhookEntity, error)
}

// Pulls all pending webhooks from the database and sends them
//
// Do not call this directly/normally, this is meant for webhook handlers such as “bothooks“
// or a potential “teamhooks“ etc.
func PullPending(p WebhookPullPending) {
	// Fetch every pending bot webhook from webhook_logs
	rows, err := state.Pool.Query(state.Context, "SELECT id, entity_id, user_id, url, data, sign, bad_intent, tries FROM webhook_logs WHERE state = $1 AND entity_type = $2", WebhookSaveStatePending, p.EntityType)

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
			tries     int
		)

		err := rows.Scan(&id, &entityId, &userId, &url, &data, &sign, &badIntent, &tries)

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
		err = SendCustom(&WebhookSendState{
			Url: url,
			Sign: Secret{
				Raw: sign,
			},
			Data:      data,
			BadIntent: badIntent,
			LogID:     id,
			Tries:     tries,
			UserID:    userId,
			Entity:    entity,
		})

		if err != nil {
			state.Logger.Error(err)
		}
	}
}
