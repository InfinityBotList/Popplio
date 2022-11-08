package migrations

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

func HasMigrated(ctx context.Context, pool *pgxpool.Pool) bool {
	if !colExists(ctx, pool, "bots", "extra_links") {
		return false
	}

	// This column has been moved to extra_links at this point
	if colExists(ctx, pool, "bots", "support") {
		return false
	}

	return true
}

func XSSCheck(ctx context.Context, pool *pgxpool.Pool) {
	// get every extra_link
	rows, err := pool.Query(ctx, "SELECT bot_id, extra_links FROM bots")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var botID pgtype.Text

		var extraLinks pgtype.JSONB

		err = rows.Scan(&botID, &extraLinks)

		if err != nil {
			panic(err)
		}

		var links map[string]string

		err = extraLinks.AssignTo(&links)

		if err != nil {
			panic(err)
		}

		for k := range links {
			if links[k] == "" {
				delete(links, k)
			}

			links[k] = strings.Trim(links[k], " ")

			// Internal links are not validated
			if strings.HasPrefix(k, "_") {
				fmt.Println("Internal link found, skipping validation")
				continue
			}

			// Validate URL
			if strings.HasPrefix(links[k], "http://") {
				links[k] = strings.Replace(links[k], "http://", "https://", 1)
				continue
			}

			if strings.HasPrefix(links[k], "https://") {
				continue
			}

			fmt.Println("Invalid URL found:", k, links[k])

			if k == "Support" && !strings.Contains(links[k], " ") {
				links[k] = strings.Replace(links[k], "www", "", 1)
				if strings.HasPrefix(links[k], "discord.gg/") {
					links[k] = "https://discord.gg/" + links[k][11:]
				} else if strings.HasPrefix(links[k], "discord.com/invite/") {
					links[k] = "https://discord.gg/" + links[k][19:]
				} else if strings.HasPrefix(links[k], "discord.com/") {
					links[k] = "https://discord.gg/" + links[k][12:]
				} else {
					links[k] = "https://discord.gg/" + links[k]
				}
				fmt.Println("HOTFIX: Fixed support link to", links[k])
			} else {
				// But wait, it may be safe still
				split := strings.Split(links[k], "/")[0]
				tldLst := strings.Split(split, ".")

				if len(tldLst) > 1 && (len(tldLst[len(tldLst)-1]) == 2 || slices.Contains([]string{
					"com",
					"net",
					"org",
					"fun",
					"app",
					"dev",
					"xyz",
				}, tldLst[len(tldLst)-1])) {
					fmt.Println("Fixed found URL link to", "https://"+links[k])
					links[k] = "https://" + links[k]
				} else {
					if strings.HasPrefix(links[k], "https://") {
						continue
					}

					log.Warning("Removing invalid link: ", links[k])
					delete(links, k)
					time.Sleep(5 * time.Second)
				}
			}
		}

		_, err = pool.Exec(ctx, "UPDATE bots SET extra_links = $1 WHERE bot_id = $2", links, botID.String)

		if err != nil {
			panic(err)
		}
	}
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) {
	if HasMigrated(ctx, pool) {
		fmt.Println("Nothing to do, checking for XSS...")
		XSSCheck(ctx, pool)
		return
	}

	for i, m := range miglist {
		fmt.Println("Running migration ["+strconv.Itoa(i)+"/"+strconv.Itoa(len(miglist))+"]", m.name)
		m.fn(ctx, pool)
	}

	XSSCheck(ctx, pool)
}
