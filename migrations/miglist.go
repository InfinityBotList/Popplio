package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
)

var miglist = []migrator{
	{
		name: "add_extra_links",
		fn: func(ctx context.Context, pool *pgxpool.Pool) {
			if !tableExists(ctx, pool, "bots") {
				panic("required table bots does not exist")
			}

			if colExists(ctx, pool, "bots", "extra_links") && !colExists(ctx, pool, "bots", "support") {
				fmt.Println("Nothing to do")
				return
			}

			if colExists(ctx, pool, "bots", "extra_links") {
				_, err := pool.Exec(ctx, "ALTER TABLE bots DROP COLUMN extra_links")
				if err != nil {
					panic(err)
				}
			}

			_, err := pool.Exec(ctx, "ALTER TABLE bots ADD COLUMN extra_links jsonb NOT NULL DEFAULT '{}'")
			if err != nil {
				panic(err)
			}

			// get every website, support, donate and github link
			rows, err := pool.Query(ctx, "SELECT bot_id, website, support, github, donate FROM bots")

			if err != nil {
				panic(err)
			}

			defer rows.Close()

			for rows.Next() {
				var botID pgtype.Text
				var website, support, github, donate pgtype.Text

				err = rows.Scan(&botID, &website, &support, &github, &donate)

				if err != nil {
					panic(err)
				}

				var cols = make(map[string]string)

				if !isNone(website) {
					cols["Website"] = website.String
				}

				if !isNone(support) {
					cols["Support"] = support.String
				}

				if !isNone(github) {
					cols["Github"] = github.String
				}

				if !isNone(donate) {
					cols["Donate"] = donate.String
				}

				_, err = pool.Exec(ctx, "UPDATE bots SET extra_links = $1 WHERE bot_id = $2", cols, botID.String)

				if err != nil {
					panic(err)
				}
			}

			_, err = pool.Exec(ctx, "ALTER TABLE bots DROP COLUMN support, DROP COLUMN github, DROP COLUMN donate, DROP COLUMN website")

			if err != nil {
				panic(err)
			}
		},
	},
}
