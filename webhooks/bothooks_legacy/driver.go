// Bothooks legacy core driver to manage central processing (webhook retries and pull pending etc.)
package bothooks_legacy

import (
	"popplio/webhooks/sender"

	docs "github.com/infinitybotlist/eureka/doclib"
)

type Driver struct {
}

func (d Driver) Register() {
	docs.AddWebhook(&docs.WebhookDoc{
		Name:    "Legacy",
		Summary: "Legacy Webhooks",
		Tags: []string{
			"Webhooks",
		},
		Description: `(older) v1 webhooks. Only supports Votes`,
		Format:      WebhookDataLegacy{},
		FormatName:  "WebhookLegacyResponse",
	})
}

// Pull pending is not currently supported. TODO
func (d Driver) PullPending() *sender.WebhookPullPending {
	return nil
}
