package sender

import (
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"

	"go.uber.org/zap"
)

// webhookSendState is one delivery attempt to one webhook.
//
// It carries both the description of what is being sent and the mutable outcome
// of sending it, because every step of a delivery — the SSRF check, the request
// build, the response handling — needs to record its result against the same log
// row.
type webhookSendState struct {
	// Event is the webhook event being delivered. It is also what a Discord
	// webhook renders into an embed.
	Event *events.WebhookResponse

	// BadIntent marks this as a deliberately mis-authenticated probe.
	//
	// Popplio periodically sends a payload signed with a throwaway secret to
	// check that the endpoint actually verifies signatures. For such a probe the
	// outcomes invert: a 401/403 is success, and a 2xx means the endpoint is
	// accepting unauthenticated payloads and is treated as broken.
	BadIntent bool

	// Webhook is the configured endpoint being delivered to.
	Webhook *webhookData

	// LogID is the webhook_logs row recording this attempt.
	LogID string

	// UserID is the user whose action triggered the webhook. Delivery outcomes
	// are reported back to them as notifications.
	UserID string

	// Entity is the bot, server or team the webhook belongs to.
	Entity WebhookEntity

	// SendState is the terminal state of this attempt, set exactly once by
	// cancelSend.
	SendState string

	// ResolvedIps caches the target's addresses so the bad-intent probe does not
	// repeat the lookup — and, more importantly, so it cannot resolve to a
	// different host than the one already vetted.
	ResolvedIps []string
}

// logFields is the context every log line about this delivery carries.
func (st *webhookSendState) logFields(extra ...zap.Field) []zap.Field {
	return append([]zap.Field{
		zap.String("logID", st.LogID),
		zap.String("userID", st.UserID),
		zap.String("entityID", st.Entity.EntityID),
		zap.Bool("badIntent", st.BadIntent),
	}, extra...)
}

// cancelSend records the terminal state of this attempt against its log row.
//
// The first call wins: a delivery reaches exactly one outcome, and a second call
// means some path failed to return after already concluding, which is logged as
// a warning rather than allowed to overwrite the real result.
func (st *webhookSendState) cancelSend(saveState string) {
	if saveState != "SUCCESS" {
		state.Logger.Info("Cancelling webhook send", st.logFields()...)
	}

	if st.SendState != "" {
		state.Logger.Warn("SendState is already set", st.logFields(zap.String("sendState", st.SendState))...)
		return
	}

	st.SendState = saveState

	if st.LogID != "" {
		_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1, tries = tries + 1 WHERE id = $2", saveState, st.LogID)

		if err != nil {
			state.Logger.Error("Failed to update webhook logs with new status", st.logFields(zap.Error(err))...)
		}
	}
}

// notify tells the triggering user how their webhook fared.
//
// A notification that cannot be delivered is logged and otherwise ignored: the
// webhook delivery itself has already concluded, and failing it over an
// undeliverable notification would misreport the outcome.
func (st *webhookSendState) notify(alertType types.AlertType, title, message string) {
	err := notifications.PushNotification(st.UserID, types.Alert{
		Type:    alertType,
		Message: message,
		Title:   title,
	})

	if err != nil {
		state.Logger.Error("Failed to send notification", st.logFields(zap.Error(err))...)
	}
}

// markFailed increments the failure counter on the entity's webhooks.
//
// Once the counter passes WebhookMaximumFailedRequests the webhook stops being
// delivered to, so this is what eventually retires an endpoint that never
// recovers.
func (st *webhookSendState) markFailed() error {
	_, err := state.Pool.Exec(
		state.Context,
		"UPDATE webhooks SET failed_requests = failed_requests + 1 WHERE target_id = $1 AND target_type = $2",
		st.Entity.EntityID, st.Entity.EntityType,
	)

	return err
}
