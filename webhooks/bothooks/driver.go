// Bothooks core driver to manage central processing (webhook retries and pull pending etc.)
package bothooks

import (
	"popplio/config"
	"popplio/state"
	"popplio/webhooks/sender"

	"github.com/infinitybotlist/eureka/dovewing"
)

type Driver struct {
}

func (d Driver) Register() {

}

func (d Driver) PullPending() *sender.WebhookPullPending {
	return &sender.WebhookPullPending{
		EntityType: EntityType,
		GetEntity: func(id string) (sender.WebhookEntity, error) {
			bot, err := dovewing.GetUser(state.Context, id, state.DovewingPlatformDiscord)

			if err != nil {
				return sender.WebhookEntity{}, err
			}

			entity := sender.WebhookEntity{
				EntityID:   bot.ID,
				EntityName: bot.Username,
				EntityType: EntityType,
			}

			// TODO: Hack until legacy webhooks is truly removed
			if config.UseLegacyWebhooks(id) {
				trueVal := true
				entity.SimpleAuth = &trueVal
			}

			return entity, nil
		},
	}
}
