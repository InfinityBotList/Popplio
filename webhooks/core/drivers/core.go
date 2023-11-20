package drivers

import (
	"errors"
	"fmt"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"
	"popplio/webhooks/sender"
	"slices"

	"github.com/infinitybotlist/eureka/dovewing"
	"go.uber.org/zap"
)

// ConstructableWebhook represents the base driver interface for constructing webhooks
type ConstructableWebhook interface {
	// Construct a webhook given a user ID and target ID
	Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error)

	// The target type of this webhook
	TargetType() string

	// Whether or not the entity supports construction in the first place
	CanBeConstructed(userId, targetId string) (bool, error)

	// Whether or not the entity supports 'pull pending' (restarting webhooks on server crash)
	SupportsPullPending(userId, targetId string) (bool, error)
}

var Registry = map[string]ConstructableWebhook{}

func RegisterCoreWebhook(webhook ConstructableWebhook) {
	Registry[webhook.TargetType()] = webhook
}

// Ergonomic webhook builder
type With struct {
	UserID     string
	TargetID   string
	TargetType string
	Metadata   *events.WebhookMetadata
	Data       events.WebhookEvent
}

func Send(with With) error {
	targetTypes := with.Data.TargetTypes()
	if !slices.Contains(targetTypes, with.TargetType) {
		return errors.New("invalid event type")
	}

	cd, ok := Registry[with.TargetType]

	if !ok {
		return errors.New("target type not registered")
	}

	// Check if the entity supports construction
	supports, err := cd.CanBeConstructed(with.UserID, with.TargetID)

	if err != nil {
		return fmt.Errorf("failed to check if entity supports construction: %w", err)
	}

	if !supports {
		return nil
	}

	// Construct the webhook
	target, entity, err := cd.Construct(with.UserID, with.TargetID)

	if err != nil {
		return err
	}

	if entity == nil {
		return errors.New("failed to construct webhook entity due to no entity being returned")
	}

	if entity.EntityType != with.TargetType {
		return fmt.Errorf("entity type mismatch: expected %s, got %s", with.TargetType, entity.EntityType)
	}

	user, err := dovewing.GetUser(state.Context, with.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to fetch user via dovewing for this hook", zap.Error(err), zap.String("targetType", with.TargetType), zap.String("targetID", with.TargetID), zap.String("userID", with.UserID))
		return err
	}

	resp := &events.WebhookResponse{
		Creator:  user,
		Targets:  *target,
		Type:     with.Data.Event(),
		Data:     with.Data,
		Metadata: events.ParseWebhookMetadata(with.Metadata),
	}

	d := &sender.WebhookSendState{
		UserID: resp.Creator.ID,
		Entity: *entity,
		Event:  resp,
	}

	err = sender.Send(d)

	if err != nil {
		err = notifications.PushNotification(d.UserID, types.Alert{
			Type:    types.AlertTypeError,
			Message: fmt.Sprintf("Failed to send webhook: %s with send state %s", err.Error(), d.SendState),
			Title:   "Webhook Send Successful!",
		})

		if err != nil {
			state.Logger.Error("Error when push notification for erroring webhook", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.String("sendState", d.SendState))
		}
	}

	return err
}

// Pulls all pending webhooks from the database and sends them
//
// Do not call this directly/normally, this is handled automatically in 'core'
func PullPending(p ConstructableWebhook) error {
	targetType := p.TargetType()

	// Fetch every pending bot webhook from webhook_logs
	rows, err := state.Pool.Query(state.Context, "SELECT id, target_id, user_id, data FROM webhook_logs WHERE state = $1 AND target_type = $2 AND bad_intent = false", "PENDING", targetType)

	if err != nil {
		return fmt.Errorf("failed to fetch pending webhooks: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			id       string
			targetId string
			userId   string
			event    *events.WebhookResponse
		)

		err := rows.Scan(&id, &targetId, &userId, &event)

		if err != nil {
			state.Logger.Error("Failed to scan pending webhook", zap.Error(err))
			continue
		}

		// Check if the entity supports pulls
		supports, err := p.SupportsPullPending(userId, targetId)

		if err != nil {
			state.Logger.Error("Failed to check if entity supports pulls", zap.Error(err), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return fmt.Errorf("failed to check if entity supports pulls: %w", err)
		}

		if !supports {
			continue
		}

		_, entity, err := p.Construct(userId, targetId)

		if err != nil {
			state.Logger.Error("Failed to get entity for webhook", zap.Error(err), zap.String("entityID", targetId))
			continue
		}

		if entity.EntityType != targetType {
			return fmt.Errorf("entity type mismatch: expected %s, got %s", targetType, entity.EntityType)
		}

		// Send webhook
		err = sender.Send(&sender.WebhookSendState{
			Event:  event,
			LogID:  id,
			UserID: userId,
			Entity: *entity,
		})

		if err != nil {
			state.Logger.Error("Failed to send pending webhook", zap.Error(err), zap.String("entityID", targetId))
		}
	}

	return nil
}

func PullPendingForAll() error {
	for _, v := range Registry {
		err := PullPending(v)

		if err != nil {
			return err
		}
	}

	return nil
}
