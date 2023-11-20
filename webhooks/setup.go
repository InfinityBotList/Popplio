package webhooks

import (
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"
	_ "popplio/webhooks/events"
	_ "popplio/webhooks/hooks"

	docs "github.com/infinitybotlist/eureka/doclib"
)

// Setup code
func Setup() {
	docs.AddTag(
		"Webhooks",
		"Webhooks are a way to receive events from Infinity Bot List in real time. You can use webhooks to receive events such as new votes, new reviews, and more.",
	)

	events.RegisterAllEvents()
	go drivers.PullPendingForAll()
}
