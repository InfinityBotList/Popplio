package migrate

import "popplio/state"

// Contains the list of migrations

var (
// statusBoldErr = color.New(color.Bold, color.FgRed).PrintlnFunc()
)

var migs = []migration{
	{
		name: "Create webhook_logs",
		function: func() {
			if tableExists("webhook_logs") {
				alrMigrated()
				return
			}

			// Create webhook_logs
			_, err := state.Pool.Exec(state.Context, `CREATE TABLE IF NOT EXISTS webhook_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
		target_id TEXT NOT NULL, 
		target_type TEXT NOT NULL,
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

		},
	},
}
