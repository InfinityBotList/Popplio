// Serverhooks core driver to manage central processing (webhook retries and pull pending etc.)
package serverhooks

import (
	"popplio/state"
	"popplio/webhooks/sender"
)

type Driver struct {
}

func (d Driver) Register() {

}

func (d Driver) PullPending() *sender.WebhookPullPending {
	return &sender.WebhookPullPending{
		EntityType: EntityType,
		GetEntity: func(id string) (sender.WebhookEntity, error) {
			var name string

			err := state.Pool.QueryRow(state.Context, "SELECT name FROM servers WHERE server_id = $1", id).Scan(&name)

			if err != nil {
				return sender.WebhookEntity{}, err
			}

			return sender.WebhookEntity{
				EntityID:   id,
				EntityName: name,
				EntityType: EntityType,
			}, nil
		},
	}
}
