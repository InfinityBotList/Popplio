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

// Driver represents the base driver interface for constructing webhooks
type Driver interface {
	// Construct a webhook given a user ID and target ID
	Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error)

	// The target type of this webhook
	TargetType() string

	// Whether or not the entity supports construction in the first place
	CanBeConstructed(userId, targetId string) (bool, error)

	// Whether or not the entity supports 'pull pending' (restarting webhooks on server crash)
	//
	// Most drivers should return `true` (outside of the case of an emergency or a bug in the driver)
	SupportsPullPending(userId, targetId string) (bool, error)
}

// Stores all registered drivers
//
// Note that because this is a map, it is impossible to have
// two drivers with the same target type and it also impossible
// to determine the order in which drivers are registered
var DriverRegistry = map[string]Driver{}

func RegisterDriver(driver Driver) {
	DriverRegistry[driver.TargetType()] = driver
}

// Ergonomic webhook builder
type With struct {
	UserID     string
	TargetID   string
	TargetType string
	Metadata   *events.WebhookMetadata
	Data       events.WebhookEvent
}

// Send takes a With struct, handles the construction of the webhook, and sends it
// using sender.Send(). It also handles push notifications on success
func Send(with With) error {
	targetTypes := with.Data.TargetTypes()
	if !slices.Contains(targetTypes, with.TargetType) {
		return errors.New("invalid event type")
	}

	driver, ok := DriverRegistry[with.TargetType]

	if !ok {
		return errors.New("target type not registered")
	}

	// Check if the entity supports construction
	supports, err := driver.CanBeConstructed(with.UserID, with.TargetID)

	if err != nil {
		return fmt.Errorf("failed to check if entity supports construction: %w", err)
	}

	if !supports {
		return nil
	}

	// Construct the webhook
	target, entity, err := driver.Construct(with.UserID, with.TargetID)

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

	d := &sender.WebhookData{
		UserID: resp.Creator.ID,
		Entity: *entity,
		Event:  resp,
	}

	res, err := sender.Send(d)

	if err != nil {
		perr := notifications.PushNotification(d.UserID, types.Alert{
			Type:    types.AlertTypeError,
			Message: fmt.Sprintf("Failed to send webhooks: %s", err.Error()),
			Title:   "Webhook Send Successful!",
		})

		if perr != nil {
			state.Logger.Error("Error when push notification for erroring webhook", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Any("sendState", res.SendStates))
		}
	}

	return err
}

// Pulls all pending webhooks from the database and sends them
//
// Do not call this directly/normally, this is handled automatically in setup.go
func PullPending(p Driver) error {
	targetType := p.TargetType()

	// Fetch every pending bot webhook from webhook_logs
	rows, err := state.Pool.Query(state.Context, "SELECT id, target_id, user_id, data FROM webhook_logs WHERE state = $1 AND target_type = $2 AND bad_intent = false", "PENDING", targetType)

	if err != nil {
		return fmt.Errorf("failed to fetch pending webhooks: %w", err)
	}

	defer rows.Close()

	var eventData []struct {
		ID       string
		TargetID string
		UserID   string
		Event    *events.WebhookResponse
	}

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

		eventData = append(eventData, struct {
			ID       string
			TargetID string
			UserID   string
			Event    *events.WebhookResponse
		}{ID: id, TargetID: targetId, UserID: userId, Event: event})
	}

	for _, v := range eventData {
		state.Logger.Info("Pulled event", zap.Any("event", v.Event), zap.Bool("isTestEvent", v.Event.Metadata.Test))

		// Check if the entity supports pulls
		supports, err := p.SupportsPullPending(v.UserID, v.TargetID)

		if err != nil {
			state.Logger.Error("Failed to check if entity supports pulls", zap.Error(err), zap.String("targetId", v.TargetID), zap.String("targetType", targetType))
			return fmt.Errorf("failed to check if entity supports pulls: %w", err)
		}

		if !supports {
			continue
		}

		_, entity, err := p.Construct(v.UserID, v.TargetID)

		if err != nil {
			state.Logger.Error("Failed to get entity for webhook", zap.Error(err), zap.String("entityID", v.TargetID))
			continue
		}

		if entity.EntityType != targetType {
			return fmt.Errorf("entity type mismatch: expected %s, got %s", targetType, entity.EntityType)
		}

		// Send webhook
		_, err = sender.Send(&sender.WebhookData{
			Event:  v.Event,
			LogID:  v.ID,
			UserID: v.UserID,
			Entity: *entity,
		})

		if errors.Is(err, sender.ErrNoWebhooks) {
			_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $1 WHERE id = $2", "NO_WEBHOOKS", v.ID)

			if err != nil {
				state.Logger.Error("Failed to update webhook state", zap.Error(err), zap.String("entityID", v.TargetID))
				continue
			}
		}

		if err != nil {
			state.Logger.Error("Failed to send pending webhook", zap.Error(err), zap.String("entityID", v.TargetID))
			continue
		}
	}

	return nil
}

// Pulls pending webhooks for all drivers that have been registered
func PullPendingForAll() error {
	for _, v := range DriverRegistry {
		err := PullPending(v)

		if err != nil {
			return err
		}
	}

	return nil
}
