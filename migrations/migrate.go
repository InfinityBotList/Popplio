package migrations

import (
	"context"
	"fmt"
	"popplio/state"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
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

func parseLink(key string, link string) string {
	if strings.HasPrefix(link, "http://") {
		return strings.Replace(link, "http://", "https://", 1)
	}

	if strings.HasPrefix(link, "https://") {
		return link
	}

	fmt.Println("Invalid URL found:", link)

	if key == "Support" && !strings.Contains(link, " ") {
		link = strings.Replace(link, "www", "", 1)
		if strings.HasPrefix(link, "discord.gg/") {
			link = "https://discord.gg/" + link[11:]
		} else if strings.HasPrefix(link, "discord.com/invite/") {
			link = "https://discord.gg/" + link[19:]
		} else if strings.HasPrefix(link, "discord.com/") {
			link = "https://discord.gg/" + link[12:]
		} else {
			link = "https://discord.gg/" + link
		}
		fmt.Println("HOTFIX: Fixed support link to", link)
		return link
	} else {
		// But wait, it may be safe still
		split := strings.Split(link, "/")[0]
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
			fmt.Println("Fixed found URL link to", "https://"+link)
			return "https://" + link
		} else {
			if strings.HasPrefix(link, "https://") {
				return link
			}

			state.Logger.Warn("Removing invalid link: ", link)
			time.Sleep(5 * time.Second)
			return ""
		}
	}
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
			links[k] = parseLink(k, links[k])

			fmt.Println("Parsed link for", k, "is", links[k])

			if links[k] == "" {
				delete(links, k)
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
