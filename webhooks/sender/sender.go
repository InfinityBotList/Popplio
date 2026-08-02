// Package sender delivers webhook payloads to their destination.
//
// It handles what delivery involves beyond an HTTP POST: signing and
// encrypting the payload for the endpoint's secret, retrying with backoff,
// refusing to connect to private network addresses, and recording each
// attempt so users can see why a webhook is failing.
package sender

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	rand2 "math/rand"
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"
	"popplio/webhooks/core/utils"
	"slices"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Represents a internal webhook to fanout
type webhookData struct {
	ID             string   `db:"id"`
	Secret         string   `db:"secret"`
	Url            string   `db:"url"`
	Broken         bool     `db:"broken"`
	FailedRequests int      `db:"failed_requests"`
	SimpleAuth     bool     `db:"simple_auth"`
	EventWhitelist []string `db:"event_whitelist"`
}

var (
	wdColsArr = db.GetCols(webhookData{})
	wdCols    = strings.Join(wdColsArr, ",")

	ErrNoWebhooks                = errors.New("no webhooks found")
	WebhookMaximumFailedRequests = 20 // Be very lenient
)

// An abstraction over an entity whether that be a bot/team/server
type WebhookEntity struct {
	// the id of the webhook's target
	EntityID string

	// the entity type
	EntityType string

	// the name of the webhook's target
	EntityName string

	// Override whether or not the authentication is 'simple' (no auth header) or not
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

	dataBytes, err := jsonimpl.Marshal(d.Event)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Send to each webhook
	var webhErrors map[string]error
	var sendStates = make(map[string]string)
	for _, webhook := range webhooks {
		if webhook.Broken || webhook.FailedRequests >= WebhookMaximumFailedRequests {
			continue
		}

		// Basic checks
		if len(webhook.EventWhitelist) > 0 {
			// Check if event is whitelisted
			if !slices.Contains(webhook.EventWhitelist, d.Event.Type) {
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
			sendStates[webhook.ID] = "INTERNAL_ERROR"
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
			params := d.Event.Data.CreateDiscordEmbed(d.Event.Creator, d.Event.Targets)

			err = SendDiscord(
				webhook.Url,
				prefix,
				d.Entity,
				params,
			)

			if err != nil {
				return fmt.Errorf("failed to send discord webhook: %w", err)
			}

			d.cancelSend("SUCCESS")
			return nil
		}
	}

	if err := d.resolveTarget(webhook.Url); err != nil {
		return err
	}

	// Randomly send a bad webhook with invalid auth
	if !d.BadIntent {
		if rand2.Float64() < 0.4 {
			go func() {
				var logID string
				err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (target_id, target_type, user_id, url, data, bad_intent, webhook_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", d.Entity.EntityID, d.Entity.EntityType, d.UserID, webhook.Url, data, true, webhook.ID).Scan(&logID)

				if err != nil {
					state.Logger.Error("Failed to insert webhook log", d.logFields(zap.Error(err))...)
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

	state.Logger.Info("Sending webhook", d.logFields()...)

	req, err := d.buildRequest(webhook, data)

	if err != nil {
		return err
	}

	resp, err := webhookClient.Do(req)

	if err != nil {
		state.Logger.Error("Failed to send webhook", d.logFields(zap.Error(err))...)
		d.cancelSend("REQUEST_SEND_FAILURE")
		return err
	}

	// Only read a maximum of 1kb, with timeout of 65 seconds
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if err != nil {
		body = []byte("Failed to read body: " + err.Error())
	}

	var reqHeaders = map[string]string{}
	for k, v := range req.Header {
		if len(reqHeaders) > 10 {
			break
		}

		reqHeaders[k] = strings.Join(v, ",")
	}

	var respHeaders = map[string]string{}
	for k, v := range resp.Header {
		if len(respHeaders) > 20 {
			break
		}

		respHeaders[k] = strings.Join(v, ",")
	}

	_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET response = $1, status_code = $2, request_headers = $3, response_headers = $4 WHERE id = $5", body, resp.StatusCode, reqHeaders, respHeaders, d.LogID)

	if err != nil {
		state.Logger.Error("Failed to update webhook logs with response", d.logFields(zap.Error(err))...)
		return fmt.Errorf("failed to update webhook logs with response: %w", err)
	}

	switch {
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		// Remove from DB
		d.cancelSend("WEBHOOK_404_410")

		if err := d.markFailed(); err != nil {
			state.Logger.Error("Failed to update webhook logs with response", d.logFields(zap.Error(err))...)
			return fmt.Errorf("webhook failed to validate auth and failed to remove webhook from db: %w", err)
		}

		d.notify(types.AlertTypeWarning, "Whoa!", "This bot seems to not have a working rewards system.")

		return errors.New("webhook returned not found thus removing it from the database")

	case resp.StatusCode == 401 || resp.StatusCode == 403:
		if d.BadIntent {
			// webhook auth is invalid as intended,
			d.cancelSend("SUCCESS")

			return nil
		} else {
			// webhook auth is invalid, return error
			d.cancelSend("WEBHOOK_AUTH_INVALID")

			d.notify(types.AlertTypeError, "Webhook Auth Error", "Webhook could not be securely authenticated by the bot at this time. Please try again later.")

			// Set webhook to broken
			if err := d.markFailed(); err != nil {
				return errors.New("webhook failed to validate auth and failed to mark request as failed")
			}

			return errors.New("webhook auth error:" + strconv.Itoa(resp.StatusCode))
		}

	case resp.StatusCode > 400:
		d.cancelSend("RESPONSE_" + strconv.Itoa(resp.StatusCode))

		d.notify(types.AlertTypeError, "Webhook Auth Error", fmt.Sprintf("We were unable to notify this bot: %d", resp.StatusCode))

		return errors.New("webhook returned error: " + strconv.Itoa(resp.StatusCode))

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if d.BadIntent {
			d.cancelSend("WEBHOOK_BROKEN_BAD_AUTHCODE")

			d.notify(types.AlertTypeError, "Webhook Auth Error", "This webhook does not properly handle authentication at this time.")

			// Set webhook to broken
			if err := d.markFailed(); err != nil {
				return errors.New("webhook failed to validate auth and failed to mark request as failed")
			}

			return errors.New("webhook failed to validate auth")
		}

		d.cancelSend("SUCCESS")

		d.notify(types.AlertTypeSuccess, "Webhook Send Successful!", "Successfully notified "+d.Entity.EntityName+" of this action.")
	}

	return nil
}

// Sends a webhook via discord
func SendDiscord(url, prefix string, entity WebhookEntity, params *discord.Embed) error {
	// Remove out prefix
	url = state.Config.Meta.PopplioProxy + "/" + strings.TrimPrefix(url, prefix)

	payload, err := jsonimpl.Marshal(discord.WebhookMessageCreate{
		Embeds: []discord.Embed{*params},
	})

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
