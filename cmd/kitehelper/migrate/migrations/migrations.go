package migrations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"kitehelper/common"
	"kitehelper/downloader"
	"kitehelper/migrate"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5/pgtype"
)

const cdnPath = "/silverpelt/cdn/ibl"
const cdnUrl = "https://cdn.infinitybots.gg"

// Contains the list of migrations
var migs = []migrate.Migration{
	{
		ID:   "create_webhook_logs",
		Name: "Create webhook_logs",
		HasMigrated: func(pool *common.SandboxPool) error {
			if tableExists(pool, "webhook_logs") {
				return errors.New("table webhook_logs already exists")
			}

			return nil
		},
		Function: func(pool *common.SandboxPool) {

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
		HasMigrated: func(pool *common.SandboxPool) error {
			if !colExists(pool, "bots", "vanity") && tableExists(pool, "vanity") {
				return errors.New("table vanity already exists")
			}

			return nil
		},
		Function: func(pool *common.SandboxPool) {
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
		HasMigrated: func(pool *common.SandboxPool) error {
			if !colExists(pool, "team_members", "perms") {
				return errors.New("column perms does not exist")
			}

			return nil
		},
		Name: "Team permissions -> flags",
		Function: func(pool *common.SandboxPool) {
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
		HasMigrated: func(pool *common.SandboxPool) error {
			if tableExists(pool, "webhooks") && !colExists(pool, "bots", "webhooks") {
				return errors.New("table webhooks already exists")
			}

			return nil
		},
		Function: func(pool *common.SandboxPool) {
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
	{
		ID:   "banner_migration",
		Name: "Banner migration",
		HasMigrated: func(pool *common.SandboxPool) error {
			if os.Getenv("BANNERS_NEED_MIGRATION") != "" {
				return nil
			}

			if colExists(pool, "bots", "banner") {
				return nil
			}

			if !colExists(pool, "bots", "has_banner") {
				return errors.New("banners do not need migration")
			}

			return errors.New("banners do not need migration")
		},
		Function: func(pool *common.SandboxPool) {
			proxyUrl := os.Getenv("BANNER_PROXY_URL")
			for _, targetType := range []string{"bots", "servers", "teams"} {
				// Create banners directory
				err := os.MkdirAll(cdnPath+"/banners/"+targetType, 0755)

				if err != nil {
					panic(err)
				}

				// Fetch every banner
				var uniqueId string
				switch targetType {
				case "bots":
					uniqueId = "bot_id"
				case "servers":
					uniqueId = "server_id"
				case "teams":
					uniqueId = "id"
				}

				rows, err := pool.Query(context.Background(), "SELECT "+uniqueId+", banner FROM "+targetType+" WHERE banner IS NOT NULL AND banner != ''")

				if err != nil {
					panic(err)
				}

				var i = 0
				var failedIds = []string{}

				defer rows.Close()

				for rows.Next() {
					var id string
					var banner string

					err = rows.Scan(&id, &banner)

					if err != nil {
						panic(err)
					}

					i++

					if os.Getenv("ONLY_NEW") != "" && strings.HasPrefix(banner, cdnUrl) {
						migrate.StatusBoldYellow("Banner for", targetType, id, "already migrated, skipping")
						continue
					}

					fmt.Println(cdnUrl)

					if !strings.HasPrefix(banner, "https://") {
						migrate.StatusBoldYellow("Banner for", targetType, id, "is invalid, skipping")
						continue
					}

					if strings.HasPrefix(banner, "https://imgur.com") {
						banner = strings.ReplaceAll(banner, "https://imgur.com", "https://i.imgur.com")
					}

					if banner == "https://i.imgur.com/lNdMzuW.png" {
						migrate.StatusBoldYellow("Banner for", targetType, id, "is a default banner")
						continue
					}

					// Check for imgur, that needs a proxied download
					if strings.HasPrefix(banner, "https://i.imgur.com/") {
						migrate.StatusBoldYellow("IMGUR BANNER, initial: " + banner)
						bannerHash := strings.TrimPrefix(banner, "https://i.imgur.com/")
						migrate.StatusBoldYellow("IMGUR BANNER hash1: " + bannerHash)
						bannerHash = strings.TrimPrefix(bannerHash, "a/")
						migrate.StatusBoldYellow("IMGUR BANNER hash2: " + bannerHash)
						banner = "https://i.imgur.com/" + bannerHash

						bannerCheck := common.UserInput("Please validate this, input 'gone' if the banner is gone: " + banner)

						if bannerCheck == "gone" {
							migrate.StatusBoldBlue("OK setting banner to null")
							continue
						}

						if !strings.Contains(bannerHash, ".") {
							ext := common.UserInput("Please enter the file extension for the banner for " + targetType + " " + id + " (e.g. png, jpg, gif) for " + banner + " as it does not contain a file extension")
							banner = banner + "." + ext
						}

						if proxyUrl == "" {
							migrate.StatusBoldYellow("Banner for", targetType, id, "("+banner+") is hosted on imgur, continuing will require a proxy set up on another device?")

							if !common.UserInputBoolean("Do you want to continue?") {
								failedIds = append(failedIds, id)
								continue
							}

							proxyUrl = common.UserInput("Please enter the proxy url to use?")

							if !strings.HasPrefix(proxyUrl, "http://") && !strings.HasPrefix(proxyUrl, "https://") {
								proxyUrl = "http://" + proxyUrl
							}
						}

						banner = proxyUrl + "/?url=" + banner
					}

					// Check if $cdnPath/banners/$botId.webp exists
					var filePath = "banners/" + targetType + "/" + id + ".webp"
					if _, err := os.Stat(cdnPath + "/" + filePath); err == nil {
						// Banner already exists, ask for user input
						if os.Getenv("ONLY_NEW") != "" {
							migrate.StatusBoldYellow("Banner for", targetType, id, "already exists [points to "+banner+"]", "("+filePath+")")
							continue
						}

						if !common.UserInputBoolean("Banner for " + targetType + " " + id + " already exists [points to " + banner + "]" + "(" + filePath + ", do you want to overwrite it?") {
							continue
						}
					}

					migrate.StatusBoldBlue("Waiting 1 seconds to avoid rate limiting")
					time.Sleep(1 * time.Second)

					migrate.StatusBoldBlue("Migrating banner for", targetType, id, "["+banner+"]")

					// First retrieve just the http headers of a CDN request without downloading the body
					resp, err := http.Head(banner)

					if err != nil {
						panic(err)
					}

					if resp.StatusCode != 200 {
						if resp.StatusCode == 403 || resp.StatusCode == 404 {
							migrate.StatusBoldBlue("OK, setting banner to null")
							continue
						}

						failedIds = append(failedIds, id)
						migrate.StatusBoldYellow("Banner for", targetType, id, "is invalid, got status code", resp.StatusCode)
						continue
					}

					// Check if the content type is an image
					if !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
						migrate.StatusBoldYellow("Banner for", targetType, id, "is invalid, got content type", resp.Header.Get("Content-Type"))
						continue
					}

					fileExtension := strings.Split(resp.Header.Get("Content-Type"), "/")[1]

					// Download the banner
					bannerData, err := downloader.DownloadFileWithProgress(banner)

					if err != nil {
						panic(err)
					}

					// Save the banner to the CDN
					var preconv = "banners/" + targetType + "/preconv_" + id + "." + fileExtension

					err = os.WriteFile(cdnPath+"/"+preconv, bannerData, 0644)

					if err != nil {
						panic(err)
					}

					// Convert to webp
					cmd := []string{"cwebp", "-q", "100", cdnPath + "/" + preconv, "-o", cdnPath + "/" + filePath}

					if fileExtension == "gif" {
						cmd = []string{"gif2webp", "-q", "100", "-m", "3", cdnPath + "/" + preconv, "-o", cdnPath + "/" + filePath, "-v"}
					}

					cmdExec := exec.Command(cmd[0], cmd[1:]...)
					cmdExec.Stdout = os.Stdout
					cmdExec.Stderr = os.Stderr
					cmdExec.Env = os.Environ()

					err = cmdExec.Run()

					if err != nil {
						panic(err)
					}

					// Delete the original file
					err = os.Remove(cdnPath + "/" + preconv)

					if err != nil {
						panic(err)
					}
				}

				if i == 0 {
					migrate.StatusBoldBlue("No banners to migrate for", targetType)
				}

				if len(failedIds) > 0 {
					fmt.Println("Failed to migrate banners for", targetType, "with ids", strings.Join(failedIds, ","))
				}
			}
		},
	},
	{
		ID:   "migrate_team_avatars",
		Name: "Migrate team avatars",
		HasMigrated: func(pool *common.SandboxPool) error {
			if os.Getenv("TEAMAVATARS_NEED_MIGRATION") != "" {
				return nil
			}

			if colExists(pool, "teams", "avatar") {
				return nil
			}

			return errors.New("avatars do not need migration")
		},
		Function: func(pool *common.SandboxPool) {
			proxyUrl := os.Getenv("BANNER_PROXY_URL")
			// Create banners directory
			err := os.MkdirAll(cdnPath+"/avatars/teams", 0755)

			if err != nil {
				panic(err)
			}

			rows, err := pool.Query(context.Background(), "SELECT id, avatar FROM teams WHERE avatar IS NOT NULL AND avatar != ''")

			if err != nil {
				panic(err)
			}

			var i = 0
			var failedIds = []string{}

			defer rows.Close()

			for rows.Next() {
				var id string
				var avatar string

				err = rows.Scan(&id, &avatar)

				if err != nil {
					panic(err)
				}

				i++

				if os.Getenv("ONLY_NEW") != "" && strings.HasPrefix(avatar, cdnUrl) {
					migrate.StatusBoldYellow("Avatar for team", id, "already migrated, skipping")
					continue
				}

				fmt.Println(cdnUrl)

				if !strings.HasPrefix(avatar, "https://") {
					migrate.StatusBoldYellow("Avatar for team", id, "is invalid, skipping")
					continue
				}

				if strings.HasPrefix(avatar, "https://imgur.com") {
					avatar = strings.ReplaceAll(avatar, "https://imgur.com", "https://i.imgur.com")
				}

				if avatar == "https://cdn.discordapp.com/embed/avatars/0.png" {
					migrate.StatusBoldYellow("Avatar for team", id, "is a default avatar")
					continue
				}

				// Check for imgur, that needs a proxied download
				if strings.HasPrefix(avatar, "https://i.imgur.com/") {
					migrate.StatusBoldYellow("IMGUR AVATAR, initial: " + avatar)
					avatarHash := strings.TrimPrefix(avatar, "https://i.imgur.com/")
					migrate.StatusBoldYellow("IMGUR AVATAR hash1: " + avatarHash)
					avatarHash = strings.TrimPrefix(avatarHash, "a/")
					migrate.StatusBoldYellow("IMGUR AVATAR hash2: " + avatarHash)
					avatar = "https://i.imgur.com/" + avatarHash

					avatarCheck := common.UserInput("Please validate this, input 'gone' if the avatar is gone: " + avatar)

					if avatarCheck == "gone" {
						migrate.StatusBoldBlue("OK, setting avatar to null")
						time.Sleep(1 * time.Second)

						continue
					}

					if !strings.Contains(avatarHash, ".") {
						ext := common.UserInput("Please enter the file extension for the avatar for team" + " " + id + " (e.g. png, jpg, gif) for " + avatar + " as it does not contain a file extension")
						avatar = avatar + "." + ext
					}

					if proxyUrl == "" {
						migrate.StatusBoldYellow("Avatar for team", id, "("+avatar+") is hosted on imgur, continuing will require a proxy set up on another device?")

						if !common.UserInputBoolean("Do you want to continue?") {
							failedIds = append(failedIds, id)
							continue
						}

						proxyUrl = common.UserInput("Please enter the proxy url to use?")

						if !strings.HasPrefix(proxyUrl, "http://") && !strings.HasPrefix(proxyUrl, "https://") {
							proxyUrl = "http://" + proxyUrl
						}
					}

					avatar = proxyUrl + "/?url=" + avatar
				}

				// Check if $cdnPath/avatars/teams/$id.webp exists
				var filePath = "avatars/teams/" + id + ".webp"
				if _, err := os.Stat(cdnPath + "/" + filePath); err == nil {
					// Banner already exists, ask for user input
					if os.Getenv("ONLY_NEW") != "" {
						migrate.StatusBoldYellow("Avatar for team", id, "already exists [points to "+avatar+"]", "("+filePath+")")
						continue
					}

					if !common.UserInputBoolean("Avatar for team " + id + " already exists [points to " + avatar + "]" + "(" + filePath + ", do you want to overwrite it?") {
						continue
					}
				}

				migrate.StatusBoldBlue("Waiting 1 seconds to avoid rate limiting")
				time.Sleep(1 * time.Second)

				migrate.StatusBoldBlue("Migrating avatar for team", id, "["+avatar+"]")

				// First retrieve just the http headers of a CDN request without downloading the body
				resp, err := http.Head(avatar)

				if err != nil {
					panic(err)
				}

				if resp.StatusCode != 200 {
					if resp.StatusCode == 403 || resp.StatusCode == 404 {
						migrate.StatusBoldBlue("OK, setting avatar to null")
						continue
					}

					failedIds = append(failedIds, id)
					migrate.StatusBoldYellow("Avatar for team", id, "is invalid, got status code", resp.StatusCode)
					continue
				}

				// Check if the content type is an image
				if !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
					migrate.StatusBoldYellow("Avatar for team", id, "is invalid, got content type", resp.Header.Get("Content-Type"))
					continue
				}

				fileExtension := strings.Split(resp.Header.Get("Content-Type"), "/")[1]

				// Download the banner
				bannerData, err := downloader.DownloadFileWithProgress(avatar)

				if err != nil {
					panic(err)
				}

				// Save the banner to the CDN
				var preconv = "avatars/teams/preconv_" + id + "." + fileExtension

				err = os.WriteFile(cdnPath+"/"+preconv, bannerData, 0644)

				if err != nil {
					panic(err)
				}

				// Convert to webp
				cmd := []string{"cwebp", "-q", "100", cdnPath + "/" + preconv, "-o", cdnPath + "/" + filePath}

				if fileExtension == "gif" {
					cmd = []string{"gif2webp", "-q", "100", "-m", "3", cdnPath + "/" + preconv, "-o", cdnPath + "/" + filePath, "-v"}
				}

				cmdExec := exec.Command(cmd[0], cmd[1:]...)
				cmdExec.Stdout = os.Stdout
				cmdExec.Stderr = os.Stderr
				cmdExec.Env = os.Environ()

				err = cmdExec.Run()

				if err != nil {
					panic(err)
				}

				// Delete the original file
				err = os.Remove(cdnPath + "/" + preconv)

				if err != nil {
					panic(err)
				}
			}

			if i == 0 {
				migrate.StatusBoldBlue("No avatars to migrate")
			}

			if len(failedIds) > 0 {
				fmt.Println("Failed to migrate team avatars with ids", strings.Join(failedIds, ","))
			}
		},
	},
	{
		ID:   "create_vanity_teams",
		Name: "Create vanity for teams",
		HasMigrated: func(pool *common.SandboxPool) error {
			if colExists(pool, "teams", "vanity_ref") {
				return errors.New("table vanity_ref already exists")
			}

			return nil
		},
		Function: func(pool *common.SandboxPool) {
			tx, err := pool.Begin(ctx)

			if err != nil {
				panic(err)
			}

			// Add column vanity_ref to teams
			err = tx.Exec(context.Background(), "ALTER TABLE teams ADD COLUMN vanity_ref UUID REFERENCES vanity(itag)")

			if err != nil {
				panic(err)
			}

			// Fetch all team ids and names
			rows, err := tx.Query(context.Background(), "SELECT id, name FROM teams")

			if err != nil {
				panic(err)
			}

			createdVanities := map[string]string{}
			for rows.Next() {
				var id string
				var name string

				err = rows.Scan(&id, &name)

				if err != nil {
					panic(err)
				}

				vanity := strings.ToLower(name)

				var repl = [][2]string{
					{" ", "-"},
					{"_", "-"},
					{".", ""},
				}

				for _, r := range repl {
					vanity = strings.ReplaceAll(vanity, r[0], r[1])
				}

				createdVanities[id] = vanity
			}

			rows.Close()

			for id, vanity := range createdVanities {
				var count int64
				err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM vanity WHERE code = $1", vanity).Scan(&count)

				if err != nil {
					panic(err)
				}

				for count > 0 {
					newVanity := vanity + "-" + crypto.RandString(8)

					var nc int64
					err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM vanity WHERE code = $1", newVanity).Scan(&nc)

					if err != nil {
						panic(err)
					}

					if nc == 0 {
						vanity = newVanity
						break
					}
				}

				migrate.StatusBoldBlue("Setting vanity for team", id, "to", vanity)

				// Insert into vanity
				var itag pgtype.UUID
				err = tx.QueryRow(context.Background(), "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, $2, $3) RETURNING itag", id, "team", vanity).Scan(&itag)

				if err != nil {
					panic(err)
				}

				// Update teams
				err = tx.Exec(context.Background(), "UPDATE teams SET vanity_ref = $1 WHERE id = $2", itag, id)

				if err != nil {
					panic(err)
				}
			}

			var nullCount int64
			err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM teams WHERE vanity_ref IS NULL").Scan(&nullCount)

			if err != nil {
				panic(err)
			}

			if nullCount > 0 {
				panic(fmt.Sprintf("nullCount > 0: %d", nullCount))
			}

			// Set vanity_ref to not null
			err = tx.Exec(context.Background(), "ALTER TABLE teams ALTER COLUMN vanity_ref SET NOT NULL")

			if err != nil {
				panic(err)
			}

			err = tx.Commit(ctx)

			if err != nil {
				panic(err)
			}
		},
	},
	{
		ID:   "migrate_bot_avatars",
		Name: "Migrate bot avatars",
		HasMigrated: func(pool *common.SandboxPool) error {
			if os.Getenv("BOTAVATARS_NEED_MIGRATION") != "" {
				return nil
			}

			return errors.New("avatars do not need migration")
		},
		Function: func(pool *common.SandboxPool) {
			// Create bots directory
			err := os.MkdirAll(cdnPath+"/avatars/bots", 0755)

			if err != nil {
				panic(err)
			}

			rows, err := pool.Query(migrate.Ctx, "SELECT bots.bot_id, internal_user_cache__discord.avatar FROM bots LEFT JOIN internal_user_cache__discord ON bots.bot_id = internal_user_cache__discord.id")

			if err != nil {
				panic(err)
			}

			var i = 0
			var failedIds = []string{}

			defer rows.Close()

			for rows.Next() {
				var id string
				var avatar string

				err = rows.Scan(&id, &avatar)

				if err != nil {
					panic(err)
				}

				i++

				fmt.Println(cdnUrl)

				if strings.HasPrefix(avatar, "https://cdn.discordapp.com/embed/avatars/") {
					migrate.StatusBoldYellow("Avatar for bot", id, "is a default avatar")
					continue
				}

				// Check if $cdnPath/avatars/bots/$id.webp exists
				var filePath = "avatars/bots/" + id + ".webp"
				if _, err := os.Stat(cdnPath + "/" + filePath); err == nil {
					// Banner already exists, ask for user input
					if os.Getenv("ONLY_NEW") != "" {
						migrate.StatusBoldYellow("Avatar for bot", id, "already exists [points to "+avatar+"]", "("+filePath+")")
						continue
					}

					if !common.UserInputBoolean("Avatar for bot " + id + " already exists [points to " + avatar + "]" + "(" + filePath + ", do you want to overwrite it?") {
						continue
					}
				}

				migrate.StatusBoldBlue("Waiting 1 seconds to avoid rate limiting")
				time.Sleep(1 * time.Second)

				migrate.StatusBoldBlue("Migrating avatar for bot", id, "["+avatar+"]")

				// First retrieve just the http headers of a CDN request without downloading the body
				resp, err := http.Head(avatar)

				if err != nil {
					panic(err)
				}

				if resp.StatusCode != 200 {
					if resp.StatusCode == 403 || resp.StatusCode == 404 {
						migrate.StatusBoldBlue("OK, setting avatar to null")
						continue
					}

					failedIds = append(failedIds, id)
					migrate.StatusBoldYellow("Avatar for bot", id, "is invalid, got status code", resp.StatusCode)
					continue
				}

				// Check if the content type is an image
				if !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
					migrate.StatusBoldYellow("Avatar for bot", id, "is invalid, got content type", resp.Header.Get("Content-Type"))
					continue
				}

				fileExtension := strings.Split(resp.Header.Get("Content-Type"), "/")[1]

				// Download the banner
				bannerData, err := downloader.DownloadFileWithProgress(avatar)

				if err != nil {
					panic(err)
				}

				// Save the banner to the CDN
				var preconv = "avatars/bots/preconv_" + id + "." + fileExtension

				err = os.WriteFile(cdnPath+"/"+preconv, bannerData, 0644)

				if err != nil {
					panic(err)
				}

				// Convert to webp
				cmd := []string{"cwebp", "-q", "100", cdnPath + "/" + preconv, "-o", cdnPath + "/" + filePath}

				if fileExtension == "gif" {
					cmd = []string{"gif2webp", "-q", "100", "-m", "3", cdnPath + "/" + preconv, "-o", cdnPath + "/" + filePath, "-v"}
				}

				cmdExec := exec.Command(cmd[0], cmd[1:]...)
				cmdExec.Stdout = os.Stdout
				cmdExec.Stderr = os.Stderr
				cmdExec.Env = os.Environ()

				err = cmdExec.Run()

				if err != nil {
					panic(err)
				}

				// Delete the original file
				err = os.Remove(cdnPath + "/" + preconv)

				if err != nil {
					panic(err)
				}
			}

			if i == 0 {
				migrate.StatusBoldBlue("No avatars to migrate")
			}

			if len(failedIds) > 0 {
				fmt.Println("Failed to migrate bot avatars with ids", strings.Join(failedIds, ","))
			}
		},
	},
	{
		ID:   "migrate_bots_to_teams",
		Name: "Migrate bot to teams",
		HasMigrated: func(pool *common.SandboxPool) error {
			if os.Getenv("MIGRATE_BOTS_TO_TEAMS_TOKEN") != "" {
				return nil
			}

			return errors.New("this can only be run with MIGRATE_BOTS_TO_TEAMS_TOKEN set to a valid token")
		},
		Function: func(pool *common.SandboxPool) {
			discordSess, err := common.NewDiscordSession(os.Getenv("MIGRATE_BOTS_TO_TEAMS_TOKEN"))

			if err != nil {
				panic(err)
			}

			tx, err := pool.Begin(ctx)

			if err != nil {
				panic(err)
			}

			defer tx.Rollback(ctx)

			rows, err := pool.Query(context.Background(), "SELECT bot_id, owner FROM bots WHERE owner IS NOT NULL")

			if err != nil {
				panic(err)
			}

			defer rows.Close()

			var botOwnerMap = map[string]string{}
			for rows.Next() {
				var botId string
				var owner string

				err = rows.Scan(&botId, &owner)

				if err != nil {
					panic(err)
				}

				botOwnerMap[botId] = owner
			}

			rows.Close()

			for botId, ownerId := range botOwnerMap {
				// Create new team with bots name and add the bot to it
				var botObj *discordgo.User

				if len(discordSess.State.Guilds) == 0 {
					panic("No guilds found")
				}

				for _, g := range discordSess.State.Guilds {
					member, err := discordSess.State.Member(g.ID, botId)

					if errors.Is(err, discordgo.ErrStateNotFound) {
						continue
					}

					if err != nil {
						panic(err)
					}

					botObj = member.User
					break
				}

				if botObj == nil {
					fmt.Println("Bot ", botId, "not found in any guilds")

					botObj, err = discordSess.User(ownerId)

					if err != nil {
						panic(err)
					}
				}

				teamName := botObj.Username
				teamId := uuid.New()

				// Create vanity
				var vanityItag string
				err = tx.QueryRow(context.Background(), "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, $2, $3) RETURNING itag", teamId, "team", teamName+crypto.RandString(8)).Scan(&vanityItag)

				if err != nil {
					panic(err)
				}

				// Create team
				err = tx.Exec(context.Background(), "INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'automigrate')", teamId, teamName, vanityItag)

				if err != nil {
					panic(err)
				}

				// Add owner to team
				err = tx.Exec(context.Background(), "INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'automigrate')", teamId, ownerId, []string{"global.*"})

				if err != nil {
					panic(err)
				}

				// Update bot
				err = tx.Exec(context.Background(), "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", teamId, botId)

				if err != nil {
					panic(err)
				}
			}

			err = tx.Commit(ctx)

			if err != nil {
				panic(err)
			}
		},
	},
}

func init() {
	migrate.AddMigrations(migs)
}
