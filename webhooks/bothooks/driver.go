// Bothooks core driver to manage central processing (webhook retries and pull pending etc.)
package bothooks

import (
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
			bot, err := dovewing.GetDiscordUser(state.Context, id)

			if err != nil {
				return sender.WebhookEntity{}, err
			}

			return sender.WebhookEntity{
				EntityID:   bot.ID,
				EntityName: bot.Username,
				EntityType: EntityType,
				DeleteWebhook: func() error {
					_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", bot.ID)

					if err != nil {
						return err
					}

					return nil
				},
			}, nil
		},
		SupportsPulls: func(id string) (bool, error) {
			// Bots only support pulls if webhooks_v2 is enabled
			var webhooksV2 bool

			err := state.Pool.QueryRow(state.Context, "SELECT webhooks_v2 FROM bots WHERE bot_id = $1", id).Scan(&webhooksV2)

			if err != nil {
				return false, err
			}

			return webhooksV2, nil
		},
	}
}
