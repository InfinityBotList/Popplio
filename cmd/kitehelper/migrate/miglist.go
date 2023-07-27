package migrate

import (
	"context"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Contains the list of migrations

var (
// statusBoldErr = color.New(color.Bold, color.FgRed).PrintlnFunc()
)

var migs = []migration{
	{
		name: "Create webhook_logs",
		function: func(pool *pgxpool.Pool) {
			if tableExists("webhook_logs") {
				alrMigrated()
				return
			}

			// Create webhook_logs
			_, err := pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS webhook_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
		target_id TEXT NOT NULL, 
		target_type TEXT NOT NULL,
		user_id TEXT NOT NULL REFERENCES users(user_id), 
		url TEXT NOT NULL, 
		data JSONB NOT NULL, 
		bad_intent BOOLEAN NOT NULL, 
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
		state TEXT NOT NULL DEFAULT 'PENDING', 
		tries INTEGER NOT NULL DEFAULT 0, 
		last_try TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	)`)

			if err != nil {
				panic(err)
			}

		},
	},
	{
		name: "Create vanity",
		function: func(pool *pgxpool.Pool) {
			if !colExists("bots", "vanity") {
				alrMigrated()
				return
			}

			// Fetch all bot vanities
			rows, err := pool.Query(context.Background(), "SELECT bot_id, vanity FROM bots")

			if err != nil {
				panic(err)
			}

			// Add column vanity_ref to bots
			_, err = pool.Exec(context.Background(), "ALTER TABLE bots ADD COLUMN vanity_ref UUID REFERENCES vanity(itag)")

			if err != nil {
				panic(err)
			}

			defer rows.Close()

			for rows.Next() {
				var botId string
				var vanity string

				err = rows.Scan(&botId, &vanity)

				if err != nil {
					panic(err)
				}

				statusBoldBlue("Migrating vanity for bot", botId)

				// Insert into vanity
				var itag pgtype.UUID
				err = pool.QueryRow(context.Background(), "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, $2, $3) RETURNING itag", botId, "bot", vanity).Scan(&itag)

				if err != nil {
					panic(err)
				}

				// Update bots
				_, err = pool.Exec(context.Background(), "UPDATE bots SET vanity_ref = $1 WHERE bot_id = $2", itag, botId)

				if err != nil {
					panic(err)
				}
			}

			// Set vanity_ref to not null
			_, err = pool.Exec(context.Background(), "ALTER TABLE bots ALTER COLUMN vanity_ref SET NOT NULL")

			if err != nil {
				panic(err)
			}
		},
	},
}
