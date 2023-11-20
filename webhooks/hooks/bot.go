package hooks

import (
	"fmt"
	"popplio/state"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"
	"popplio/webhooks/sender"

	"github.com/infinitybotlist/eureka/dovewing"
)

type BotDriver struct{}

func (bd BotDriver) TargetType() string {
	return "bot"
}

func (bd BotDriver) Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error) {
	bot, err := dovewing.GetUser(state.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch bot via dovewing for this bothook: %w, botid=%s", err, id)
	}

	targets := events.Target{
		Bot: bot,
	}
	entity := sender.WebhookEntity{
		EntityID:   bot.ID,
		EntityName: bot.Username,
		EntityType: bd.TargetType(),
	}

	return &targets, &entity, nil
}

func (bd BotDriver) CanBeConstructed(userId, targetId string) (bool, error) {
	return true, nil
}

func (bd BotDriver) SupportsPullPending(userId, targetId string) (bool, error) {
	return true, nil
}

func init() {
	drivers.RegisterCoreWebhook(BotDriver{})
}
