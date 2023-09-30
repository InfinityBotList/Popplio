package migrations

import (
	"context"
	"errors"
	"strings"

	"kitehelper/migrate"

	"github.com/jackc/pgx/v5/pgtype"
)

// Contains the list of migrations
var migs = []migrate.Migration{
	{
		ID:   "create_webhook_logs",
		Name: "Create webhook_logs",
		HasMigrated: func(pool *migrate.SandboxPool) error {
			if tableExists(pool, "webhook_logs") {
				return errors.New("table webhook_logs already exists")
			}

			return nil
		},
		Function: func(pool *migrate.SandboxPool) {

			// Create webhook_logs
			err := pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS webhook_logs (
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
		ID:   "create_vanity",
		Name: "Create vanity",
		HasMigrated: func(pool *migrate.SandboxPool) error {
			if !colExists(pool, "bots", "vanity") && tableExists(pool, "vanity") {
				return errors.New("table vanity already exists")
			}

			return nil
		},
		Function: func(pool *migrate.SandboxPool) {
			// Fetch all bot vanities
			rows, err := pool.Query(context.Background(), "SELECT bot_id, vanity FROM bots")

			if err != nil {
				panic(err)
			}

			// Add column vanity_ref to bots
			err = pool.Exec(context.Background(), "ALTER TABLE bots ADD COLUMN vanity_ref UUID REFERENCES vanity(itag)")

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

				migrate.StatusBoldBlue("Migrating vanity for bot", botId)

				// Insert into vanity
				var itag pgtype.UUID
				err = pool.QueryRow(context.Background(), "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, $2, $3) RETURNING itag", botId, "bot", vanity).Scan(&itag)

				if err != nil {
					panic(err)
				}

				// Update bots
				err = pool.Exec(context.Background(), "UPDATE bots SET vanity_ref = $1 WHERE bot_id = $2", itag, botId)

				if err != nil {
					panic(err)
				}
			}

			// Set vanity_ref to not null
			err = pool.Exec(context.Background(), "ALTER TABLE bots ALTER COLUMN vanity_ref SET NOT NULL")

			if err != nil {
				panic(err)
			}
		},
	},
	{
		ID: "team_permissions_v2",
		HasMigrated: func(pool *migrate.SandboxPool) error {
			if !colExists(pool, "team_members", "perms") {
				return errors.New("column perms does not exist")
			}

			return nil
		},
		Name: "Team permissions -> flags",
		Function: func(pool *migrate.SandboxPool) {
			// Fetch every team member permission
			pmap := map[string]string{
				"EDIT_BOT_SETTINGS":            "bot.edit",
				"ADD_NEW_BOTS":                 "bot.add",
				"RESUBMIT_BOTS":                "bot.resubmit",
				"CERTIFY_BOTS":                 "bot.request_cert",
				"VIEW_EXISTING_BOT_TOKENS":     "bot.view_api_tokens",
				"RESET_BOT_TOKEN":              "bot.reset_api_tokens",
				"EDIT_BOT_WEBHOOKS":            "bot.edit_webhooks",
				"TEST_BOT_WEBHOOKS":            "bot.test_webhooks",
				"SET_BOT_VANITY":               "bot.set_vanity",
				"DELETE_BOTS":                  "bot.delete",
				"EDIT_TEAM_INFO":               "team.edit",
				"ADD_TEAM_MEMBERS":             "team_member.add",
				"EDIT_TEAM_MEMBER_PERMISSIONS": "team_member.edit",
				"REMOVE_TEAM_MEMBERS":          "team_member.remove",
				"EDIT_TEAM_WEBHOOKS":           "team.edit_webhooks",
				"OWNER":                        "global.*",
			}

			rows, err := pool.Query(context.Background(), "SELECT team_id, user_id, perms FROM team_members")

			if err != nil {
				panic(err)
			}

			defer rows.Close()

			for rows.Next() {
				var teamId string
				var userId string
				var perms []string

				err = rows.Scan(&teamId, &userId, &perms)

				if err != nil {
					panic(err)
				}

				migrate.StatusBoldBlue("Migrating team member permissions for", userId, "in team", teamId)

				// Convert perms
				var flags = []string{}

				for _, perm := range perms {
					if flag, ok := pmap[perm]; ok {
						flags = append(flags, flag)
					}
				}

				// Update team_members
				err = pool.Exec(context.Background(), "UPDATE team_members SET flags = $1 WHERE team_id = $2 AND user_id = $3", flags, teamId, userId)

				if err != nil {
					panic(err)
				}
			}
		},
	},
	{
		ID:   "migrate_webhooks",
		Name: "migrate webhooks",
		HasMigrated: func(pool *migrate.SandboxPool) error {
			if tableExists(pool, "webhooks") && !colExists(pool, "bots", "webhooks") {
				return errors.New("table webhooks already exists")
			}

			return nil
		},
		Function: func(pool *migrate.SandboxPool) {
			rows, err := pool.Query(context.Background(), "SELECT bot_id, webhook, web_auth, api_token from bots")

			if err != nil {
				panic(err)
			}

			defer rows.Close()

			for rows.Next() {
				var botId string
				var webhook pgtype.Text
				var webAuth pgtype.Text
				var apiToken string

				err = rows.Scan(&botId, &webhook, &webAuth, &apiToken)

				if err != nil {
					panic(err)
				}

				if !webhook.Valid || !strings.HasPrefix(webhook.String, "https://") {
					continue
				}

				if !webAuth.Valid {
					webAuth = pgtype.Text{
						Valid:  true,
						String: apiToken,
					}
				}

				migrate.StatusBoldBlue("Migrating webhook for botId="+botId, "webhook="+webhook.String, "webAuth="+webAuth.String)

				// Insert into webhooks
				err = pool.Exec(context.Background(), "INSERT INTO webhooks (target_id, target_type, url, secret) VALUES ($1, 'bot', $2, $3)", botId, webhook.String, webAuth.String)

				if err != nil {
					panic(err)
				}
			}
		},
	},
}

func init() {
	migrate.AddMigrations(migs)
}
