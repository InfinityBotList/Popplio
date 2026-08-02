// Package webhooks wires up Popplio's outgoing webhook system.
//
// Setup registers the documentation tag and pulls in the event and driver
// implementations for their side effects, which is what makes them
// discoverable at runtime. The pieces live in the subpackages: core/events
// defines the event types, core/drivers dispatches per target type, and
// sender performs delivery.
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

	events.RegisterAddedEvents()
	go drivers.PullPendingForAll()
}
