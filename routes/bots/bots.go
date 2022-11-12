package bots

import (
	"io"
	"math"
	"net/http"
	"os"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

const tagName = "Bots"

var (
	botColsArr = utils.GetCols(types.Bot{})
	// These are the columns of a bot
	botCols = strings.Join(botColsArr, ",")

	reviewColsArr = utils.GetCols(types.Review{})
	// These are the columns of a review
	reviewCols = strings.Join(reviewColsArr, ",")

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to bots on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/bots", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/bots/all",
			OpId:        "get_all_bots",
			Summary:     "Get All Bots",
			Description: "Gets all bots on the list.",
			Tags:        []string{tagName},
			Resp:        types.AllBots{},
		})
		r.Get("/all", func(w http.ResponseWriter, r *http.Request) {
			const perPage = 12

			page := r.URL.Query().Get("page")

			if page == "" {
				page = "1"
			}

			pageNum, err := strconv.ParseUint(page, 10, 32)

			if err != nil {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			limit := perPage
			offset := (pageNum - 1) * perPage

			rows, err := state.Pool.Query(state.Context, "SELECT "+botCols+" FROM bots ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var bots []*types.Bot

			err = pgxscan.ScanAll(&bots, rows)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var previous strings.Builder

			// More optimized string concat
			previous.WriteString(os.Getenv("SITE_URL"))
			previous.WriteString("/bots/all?page=")
			previous.WriteString(strconv.FormatUint(pageNum-1, 10))

			if pageNum-1 < 1 || pageNum == 0 {
				previous.Reset()
			}

			var count uint64

			err = state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM bots").Scan(&count)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var next strings.Builder

			next.WriteString(os.Getenv("SITE_URL"))
			next.WriteString("/bots/all?page=")
			next.WriteString(strconv.FormatUint(pageNum+1, 10))

			if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
				next.Reset()
			}

			data := types.AllBots{
				Count:    count,
				Results:  bots,
				PerPage:  perPage,
				Previous: previous.String(),
				Next:     next.String(),
			}

			bytes, err := json.Marshal(data)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		})

		docs.Route(&docs.Doc{
			Method:  "GET",
			Path:    "/bots/{id}",
			OpId:    "get_bot",
			Summary: "Get Bot",
			Description: `
Gets a bot by id or name

**Some things to note:**

-` + state.BackTick + state.BackTick + `external_source` + state.BackTick + state.BackTick + ` shows the source of where a bot came from (Metro Reviews etc etr.). If this is set to ` + state.BackTick + state.BackTick + `metro` + state.BackTick + state.BackTick + `, then ` + state.BackTick + state.BackTick + `list_source` + state.BackTick + state.BackTick + ` will be set to the metro list ID where it came from` + `
	`,
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "The bots ID, name or vanity",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.Bot{},
			Tags: []string{tagName},
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			name := chi.URLParam(r, "id")

			if name == "" {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			// Check cache, this is how we can avoid hefty ratelimits
			cache := state.Redis.Get(state.Context, "bc-"+name).Val()
			if cache != "" {
				w.Header().Add("X-Popplio-Cached", "true")
				w.Write([]byte(cache))
				return
			}

			var bot types.Bot

			var err error

			row, err := state.Pool.Query(state.Context, "SELECT "+botCols+" FROM bots WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", name)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			err = pgxscan.ScanOne(&bot, row)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			err = utils.ParseBot(state.Context, state.Pool, &bot, state.Discord, state.Redis)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			var uniqueClicks int64
			err = state.Pool.QueryRow(state.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			bot.UniqueClicks = uniqueClicks

			/* Removing or modifying fields directly in API is very dangerous as scrapers will
			 * just ignore owner checks anyways or cross-reference via another list. Also we
			 * want to respect the permissions of the owner if they're the one giving permission,
			 * blocking IPs is a better idea to this
			 */

			bytes, err := json.Marshal(bot)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			state.Redis.Set(state.Context, "bc-"+name, string(bytes), time.Minute*3)

			w.Write(bytes)
		})

		docs.Route(&docs.Doc{
			Method:  "POST",
			Path:    "/bots/stats",
			OpId:    "post_stats",
			Summary: "Post Bot Stats",
			Description: `
	This endpoint can be used to post the stats of a bot.
	
	The variation` + state.BackTick + `/bots/{bot_id}/stats` + state.BackTick + ` can also be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**
	
	**Example:**
	
	` + state.BackTick + state.BackTick + state.BackTick + `py
	import requests
	
	req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})
	
	print(req.json())
	` + state.BackTick + state.BackTick + state.BackTick + "\n\n",
			Tags:     []string{tagName},
			Req:      types.BotStatsDocs{},
			Resp:     types.ApiError{},
			AuthType: []string{"Bot"},
		})
		r.Post("/stats", func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			var id *string

			// Check token
			if r.Header.Get("Authorization") == "" {
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			} else {
				id = utils.AuthCheck(r.Header.Get("Authorization"), true)

				if id == nil {
					utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
					return
				}
			}

			defer r.Body.Close()

			var payload types.BotStats

			bodyBytes, err := io.ReadAll(r.Body)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			err = json.Unmarshal(bodyBytes, &payload)

			if err != nil {
				if r.URL.Query().Get("count") != "" {
					payload = types.BotStats{}
				} else {
					log.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(state.BadRequestStats))
					return
				}
			}

			if r.URL.Query().Get("count") != "" {
				count, err := strconv.ParseUint(r.URL.Query().Get("count"), 10, 32)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
					return
				}

				var countAny any = count

				payload.Count = &countAny
			}

			servers, shards, users := payload.GetStats()

			if servers > 0 {
				_, err = state.Pool.Exec(state.Context, "UPDATE bots SET servers = $1 WHERE bot_id = $2", servers, id)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}
			}

			if shards > 0 {
				_, err = state.Pool.Exec(state.Context, "UPDATE bots SET shards = $1 WHERE bot_id = $2", shards, id)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}
			}

			if users > 0 {
				_, err = state.Pool.Exec(state.Context, "UPDATE bots SET users = $1 WHERE bot_id = $2", users, id)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}
			}

			// Get name and vanity, delete from cache
			var name, vanity string

			state.Pool.QueryRow(state.Context, "SELECT name, vanity FROM bots WHERE bot_id = $1", id).Scan(&name, &vanity)

			// Delete from cache
			state.Redis.Del(state.Context, "bc-"+name)
			state.Redis.Del(state.Context, "bc-"+vanity)
			state.Redis.Del(state.Context, "bc-"+*id)

			w.Write([]byte(state.Success))
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/bots/{id}/seo",
			OpId:        "get_bot_seo",
			Summary:     "Get Bot SEO Info",
			Description: "Gets the minimal SEO information about a bot for embed/search purposes. Used by v4 website for meta tags",
			Resp:        types.SEO{},
			Tags:        []string{tagName},
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "The bots ID, name or vanity",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
		})
		r.Get("/{id}/seo", func(w http.ResponseWriter, r *http.Request) {
			name := chi.URLParam(r, "id")

			if name == "" {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			cache := state.Redis.Get(state.Context, "seob:"+name).Val()
			if cache != "" {
				w.Header().Add("X-Popplio-Cached", "true")
				w.Write([]byte(cache))
				return
			}

			var botId string
			var short string
			err := state.Pool.QueryRow(state.Context, "SELECT bot_id, short FROM bots WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", name).Scan(&botId, &short)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			bot, err := utils.GetDiscordUser(botId)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			bytes, err := json.Marshal(types.SEO{
				ID:       bot.ID,
				Username: bot.Username,
				Avatar:   bot.Avatar,
				Short:    short,
			})

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			state.Redis.Set(state.Context, "seob:"+name, string(bytes), time.Minute*30)

			w.Write(bytes)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/bots/{id}/reviews",
			OpId:        "get_bot_reviews",
			Summary:     "Get Bot Reviews",
			Description: "Gets the reviews of a bot by its ID, name or vanity",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "The bots ID, name or vanity",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.ReviewList{},
			Tags: []string{tagName},
		})
		r.Get("/{id}/reviews", func(w http.ResponseWriter, r *http.Request) {
			rows, err := state.Pool.Query(state.Context, "SELECT "+reviewCols+" FROM reviews WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", chi.URLParam(r, "id"))

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			var reviews []types.Review = []types.Review{}

			err = pgxscan.ScanAll(&reviews, rows)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var allReviews types.ReviewList = types.ReviewList{
				Reviews: reviews,
			}

			bytes, err := json.Marshal(allReviews)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		})
	})
}
