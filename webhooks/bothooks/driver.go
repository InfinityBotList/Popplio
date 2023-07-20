// Bothooks core driver to manage central processing (webhook retries and pull pending etc.)
package bothooks

import (
	"errors"
	"popplio/config"
	"popplio/state"
	"popplio/webhooks/sender"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

type Driver struct {
}

func (d Driver) Register() {

}

func (d Driver) PullPending() *sender.WebhookPullPending {
	return &sender.WebhookPullPending{
		EntityType: EntityType,
		GetSecret: func(id string) (sender.Secret, error) {
			var sign pgtype.Text
			err := state.Pool.QueryRow(state.Context, "SELECT web_auth FROM bots WHERE bot_id = $1", id).Scan(&sign)

			if err != nil {
				return sender.Secret{}, err
			}

			if !sign.Valid {
				return sender.Secret{}, errors.New("webhook secret is not set")
			}

			return sender.Secret{
				Raw:         sign.String,
				UseInsecure: config.UseLegacyWebhooks(id),
			}, nil
		},
		GetEntity: func(id string) (sender.WebhookEntity, error) {
			bot, err := dovewing.GetUser(state.Context, id, state.DovewingPlatformDiscord)

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
	}
}
