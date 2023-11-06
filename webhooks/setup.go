package webhooks

import (
	"popplio/webhooks/bothooks"
	"popplio/webhooks/bothooks_legacy"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"popplio/webhooks/serverhooks"
	"popplio/webhooks/teamhooks"

	docs "github.com/infinitybotlist/eureka/doclib"
)

// A webhook driver
//
// TODO: This will also be used to handle retries in the future
type WebhookDriver interface {
	PullPending() *sender.WebhookPullPending

	// If any specific setup is required for the driver, it can be done here
	Register()
}

var RegisteredDrivers = map[string]WebhookDriver{
	bothooks.EntityType:    bothooks.Driver{},
	serverhooks.EntityType: serverhooks.Driver{},
	teamhooks.EntityType:   teamhooks.Driver{},
}

// Setup code
func Setup() {
	docs.AddTag(
		"Webhooks",
		"Webhooks are a way to receive events from Infinity Bot List in real time. You can use webhooks to receive events such as new votes, new reviews, and more.",
	)

	events.RegisterAllEvents()

	for _, driver := range RegisteredDrivers {
		driver.Register()

		pullPending := driver.PullPending()

		if pullPending != nil {
			go sender.PullPending(*pullPending)
		}
	}

	legacyDocs()
}

// UNFORTUNATELY needed
func legacyDocs() {
	docs.AddWebhook(&docs.WebhookDoc{
		Name:    "Legacy",
		Summary: "Legacy Webhooks",
		Tags: []string{
			"Webhooks",
		},
		Description: `(older) v1 webhooks. Only supports Votes`,
		Format:      bothooks_legacy.WebhookDataLegacy{},
		FormatName:  "WebhookLegacyResponse",
	})
}
