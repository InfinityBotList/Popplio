package sender

import (
	"bytes"
	"context"
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
	"net"
	"net/http"
	"net/url"
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"
	"popplio/webhooks/core/utils"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var ErrNoWebhooks = errors.New("no webhooks found")

// Represents a internal webhook to fanout
type webhookData struct {
	ID             string   `db:"id"`
	Secret         string   `db:"secret"`
	Url            string   `db:"url"`
	Broken         bool     `db:"broken"`
	SimpleAuth     bool     `db:"simple_auth"`
	EventWhitelist []string `db:"event_whitelist"`
}

var (
	json      = jsoniter.ConfigCompatibleWithStandardLibrary
	wdColsArr = db.GetCols(webhookData{})
	wdCols    = strings.Join(wdColsArr, ",")
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

// Internal structure to store a webhook send state
type webhookSendState struct {
	// webhook event (used for discord webhooks)
	Event *events.WebhookResponse

	// Is it a bad intent: intentionally bad auth to trigger 401 check
	BadIntent bool

	// Webhook data
	Webhook *webhookData

	// Automatically set fields
	LogID string

	// user id that triggered the webhook
	UserID string

	// The entity itself
	Entity WebhookEntity

	// Send state, this is automatically set by Send
	SendState string

	// Internal cached list of resolved IPs for SSRF protection
	ResolvedIps []string
}

func (st *webhookSendState) cancelSend(saveState string) {
	if saveState != "SUCCESS" {
		state.Logger.Info("Cancelling webhook send", zap.String("logID", st.LogID), zap.String("userID", st.UserID), zap.String("entityID", st.Entity.EntityID), zap.Bool("badIntent", st.BadIntent))
	}

	if st.SendState != "" {
		state.Logger.Warn("SendState is already set", zap.String("logID", st.LogID), zap.String("userID", st.UserID), zap.String("entityID", st.Entity.EntityID), zap.Bool("badIntent", st.BadIntent), zap.String("sendState", st.SendState))
		return
	}

	st.SendState = saveState

	if st.LogID != "" {
		_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1, tries = tries + 1 WHERE id = $2", saveState, st.LogID)

		if err != nil {
			state.Logger.Error("Failed to update webhook logs with new status", zap.Error(err), zap.String("logID", st.LogID), zap.String("userID", st.UserID), zap.String("entityID", st.Entity.EntityID), zap.Bool("badIntent", st.BadIntent))
		}
	}
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

// External state that should be used by public function definitions
type WebhookData struct {
	// webhook event (used for discord webhooks)
	Event *events.WebhookResponse

	// Is it a bad intent: intentionally bad auth to trigger 401 check
	BadIntent bool

	// Log ID (pull pending etc)
	LogID string

	// user id that triggered the webhook
	UserID string

	// The entity itself
	Entity WebhookEntity
}

// Represents a webhook send result
type WebhookSendResult struct {
	SendStates map[string]string
}

// Creates a webhook response fanning it out to multiple webhooks if needed, retrying if needed
func Send(d *WebhookData) (*WebhookSendResult, error) {
	if !d.Entity.Validate() {
		panic("invalid webhook entity")
	}

	if d.Event == nil {
		panic("no event set in sendstate and this should never happen")
	}

	rows, err := state.Pool.Query(state.Context, "SELECT "+wdCols+" FROM webhooks WHERE target_id = $1 AND target_type = $2", d.Entity.EntityID, d.Entity.EntityType)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch webhooks: %w", err)
	}

	webhooks, err := pgx.CollectRows(rows, pgx.RowToStructByName[webhookData])

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoWebhooks
	}

	if err != nil {
		return nil, fmt.Errorf("failed to collect webhooks: %w", err)
	}

	dataBytes, err := json.Marshal(d.Event)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Send to each webhook
	var webhErrors map[string]error
	var sendStates = make(map[string]string)
	for _, webhook := range webhooks {
		if webhook.Broken {
			continue
		}

		// Basic checks
		if len(webhook.EventWhitelist) > 0 {
			// Check if event is whitelisted
			if !slices.Contains(webhook.EventWhitelist, d.Event.Type) {
				sendStates[webhook.ID] = "EVENT_NOT_WHITELISTED"
				continue
			}
		}

		// Create a log entry
		var logID string
		if d.LogID == "" {
			err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (target_id, target_type, user_id, url, data, bad_intent, webhook_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", d.Entity.EntityID, d.Entity.EntityType, d.UserID, webhook.Url, dataBytes, d.BadIntent, webhook.ID).Scan(&logID)

			if err != nil {
				if webhErrors == nil {
					webhErrors = make(map[string]error)
				}

				webhErrors[webhook.ID] = err

				continue
			}
		} else {
			logID = d.LogID
		}

		st := &webhookSendState{
			Event:     d.Event,
			BadIntent: d.BadIntent,
			Webhook:   &webhook,
			LogID:     logID,
			UserID:    d.UserID,
			Entity:    d.Entity,
		}

		err = send(st, &webhook, &dataBytes)

		if err != nil {
			if webhErrors == nil {
				webhErrors = make(map[string]error)
			}

			webhErrors[webhook.ID] = err

			continue
		}

		sendStates[webhook.ID] = st.SendState
	}

	if len(sendStates) == 0 {
		return nil, ErrNoWebhooks
	}

	if len(webhErrors) > 0 {
		var errStr = strings.Builder{}
		for url, err := range webhErrors {
			errStr.WriteString(fmt.Sprintf("%s: %s\n", url, err.Error()))
		}

		return nil, errors.New(errStr.String())
	}

	res := &WebhookSendResult{
		SendStates: sendStates,
	}

	return res, nil
}

// Internal definition for send
func send(d *webhookSendState, webhook *webhookData, pBytes *[]byte) error {
	if !d.Entity.Validate() {
		panic("invalid webhook entity")
	}

	if pBytes == nil {
		panic("pBytes is nil")
	}

	data := *pBytes

	// Unmarshal event data if no data is set
	if !d.BadIntent {
		prefix, err := utils.GetDiscordWebhookInfo(webhook.Url)

		if err != nil && !errors.Is(err, utils.ErrNotActuallyWebhook) {
			return fmt.Errorf("error while checking webhook: %w", err)
		}

		if prefix != "" && !errors.Is(err, utils.ErrNotActuallyWebhook) {
			params := d.Event.Data.CreateHookParams(d.Event.Creator, d.Event.Targets)

			err = SendDiscord(
				webhook.Url,
				prefix,
				d.Entity,
				params,
			)

			if err != nil {
				return fmt.Errorf("failed to send discord webhook: %w", err)
			}

			return nil
		}
	}

	// Resolve URL first to avoid SSRF
	if len(d.ResolvedIps) == 0 {
		url, err := url.ParseRequestURI(webhook.Url)

		if err != nil {
			d.cancelSend("INVALID_REQUEST_URL")
			return err
		}

		timeoutCtx, cancel := context.WithTimeout(state.Context, 5*time.Second)
		defer cancel()
		ip, err := net.DefaultResolver.LookupHost(timeoutCtx, url.Hostname())

		if err != nil {
			d.cancelSend("CNAME_LOOKUP_FAILURE")
			return err
		}

		d.ResolvedIps = ip
	}

	state.Logger.Info("Resolved webhook IP", zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent), zap.Strings("resolvedIp", d.ResolvedIps))
	if slices.Contains(d.ResolvedIps, "127.0.0.1") {
		d.cancelSend("LOCALHOST_URL")
		return errors.New("localhost url")
	}

	// Randomly send a bad webhook with invalid auth
	if !d.BadIntent {
		if rand2.Float64() < 0.4 {
			go func() {
				var logID string
				err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (target_id, target_type, user_id, url, data, bad_intent, webhook_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", d.Entity.EntityID, d.Entity.EntityType, d.UserID, webhook.Url, data, d.BadIntent, webhook.ID).Scan(&logID)

				if err != nil {
					state.Logger.Error("Failed to insert webhook log", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
					return
				}

				badD := &webhookSendState{
					Event:       d.Event,
					BadIntent:   true,
					Webhook:     webhook,
					UserID:      d.UserID,
					LogID:       logID,
					Entity:      d.Entity,
					ResolvedIps: d.ResolvedIps,
				}

				// Retry with bad intent
				send(badD, webhook, pBytes)
			}()
		}
	}

	state.Logger.Info("Sending webhook", zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	var secret string
	if d.BadIntent {
		secret = crypto.RandString(128)
	} else {
		secret = webhook.Secret
	}

	var req *http.Request
	var err error

	if webhook.SimpleAuth {
		req, err = http.NewRequestWithContext(state.Context, "POST", webhook.Url, bytes.NewReader(data))

		if err != nil {
			return err
		}

		req.Header.Set("Authorization", secret)
		req.Header.Set("X-Webhook-Protocol", "simple-auth")
	} else {
		// Generate HMAC token using nonce and signed header for further randomization
		nonce := crypto.RandString(16)

		keyHash := sha256.New()
		keyHash.Write([]byte(secret + nonce))

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

		postData := []byte(hex.EncodeToString(gcm.Seal(aesNonce, aesNonce, data, nil)))

		// HMAC with encrypted request body
		tok1 := Secret{Raw: secret}.Sign(postData)

		finalToken := Secret{Raw: nonce}.Sign([]byte(tok1))

		req, err = http.NewRequestWithContext(state.Context, "POST", webhook.Url, bytes.NewReader(postData))

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

	case resp.StatusCode == http.StatusTeapot || resp.StatusCode == http.StatusNotImplemented || resp.StatusCode == http.StatusServiceUnavailable:
		d.cancelSend("TEAPOT_INVALID")

		if d.Event != nil {
			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    types.AlertTypeError,
				Message: "This bot can't respond to " + d.Event.Type + " events at this time!",
				Title:   "Webhook Error",
			})

			if err != nil {
				state.Logger.Error("Failed to send notification", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Bool("badIntent", d.BadIntent))
			}
		}

		return errors.New("webhook returned teapot [unsupported event/internal error in initial processing]")

	case resp.StatusCode == 401 || resp.StatusCode == 403:
		if d.BadIntent {
			// webhook auth is invalid as intended,
			d.cancelSend("SUCCESS")

			return nil
		} else {
			// webhook auth is invalid, return error
			d.cancelSend("WEBHOOK_AUTH_INVALID")
			err = notifications.PushNotification(d.UserID, types.Alert{
				Type:    types.AlertTypeError,
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

// Sends a webhook via discord
func SendDiscord(url, prefix string, entity WebhookEntity, params *discordgo.WebhookParams) error {
	// Remove out prefix
	url = state.Config.Meta.PopplioProxy + "/" + strings.TrimPrefix(url, prefix)

	payload, err := json.Marshal(params)

	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))

	if err != nil {
		return err
	}

	for _, code := range []int{404, 401, 403, 410} {
		if resp.StatusCode == code {
			// This webhook is broken
			_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2", entity.EntityID, entity.EntityType)

			if err != nil {
				state.Logger.Error("Failed to update webhook logs with response", zap.Error(err), zap.String("entityID", entity.EntityID), zap.String("entityType", entity.EntityType), zap.Int("status", resp.StatusCode))
				return fmt.Errorf("webhook is broken (404/401/403/410) and failed to remove webhook from db: %w", err)
			}

			return errors.New("webhook is broken (404/401/403/410)")
		}
	}

	return nil
}
