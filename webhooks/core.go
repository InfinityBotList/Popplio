package webhooks

import (
	"popplio/state"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/bothooks/legacy"

	docs "github.com/infinitybotlist/doclib"
)

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
		entity_type INTEGER NOT NULL,
		user_id TEXT NOT NULL REFERENCES users(user_id), 
		url TEXT NOT NULL, 
		data JSONB NOT NULL, 
		sign TEXT NOT NULL, 
		bad_intent BOOLEAN NOT NULL, 
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
		state INTEGER NOT NULL DEFAULT 0, 
		tries INTEGER NOT NULL DEFAULT 0, 
		last_try TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	if err != nil {
		panic(err)
	}

	bothooks.Setup()
	legacy.Setup()
}
