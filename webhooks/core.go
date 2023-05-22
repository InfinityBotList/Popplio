package webhooks

import (
	"popplio/state"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"

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
	bothooks.EntityType: bothooks.Driver{},
}

// Setup code
func Setup() {
	docs.AddTag(
		"Webhooks",
		"Webhooks are a way to receive events from Infinity Bot List in real time. You can use webhooks to receive events such as new votes, new reviews, and more.",
	)
	// Create webhook_logs
	_, err := state.Pool.Exec(state.Context, `CREATE TABLE IF NOT EXISTS webhook_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
		entity_id TEXT NOT NULL, 
		entity_type TEXT NOT NULL,
		user_id TEXT NOT NULL REFERENCES users(user_id), 
		url TEXT NOT NULL, 
		data JSONB NOT NULL, 
		sign TEXT NOT NULL, 
		bad_intent BOOLEAN NOT NULL, 
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
		state TEXT NOT NULL DEFAULT 'PENDING', 
		tries INTEGER NOT NULL DEFAULT 0, 
		last_try TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		use_insecure BOOLEAN NOT NULL DEFAULT FALSE
	)`)

	if err != nil {
		panic(err)
	}

	events.Setup()

	for _, driver := range RegisteredDrivers {
		driver.Register()

		pullPending := driver.PullPending()

		if pullPending != nil {
			go sender.PullPending(*pullPending)
		}
	}
}
