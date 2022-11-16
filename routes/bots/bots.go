package bots

import (
	"io"
	"math"
	"net/http"
	"os"
	"popplio/constants"
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
)

const (
	tagName = "Bots"
	perPage = 12
)

var (
	botColsArr = utils.GetCols(types.Bot{})
	botCols    = strings.Join(botColsArr, ",")

	reviewColsArr = utils.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")

	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

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
			Description: "Gets all bots on the list. Returns a ``Index`` object",
			Tags:        []string{tagName},
			Resp:        types.AllBots{},
		})
		r.Get("/all", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			resp := make(chan types.HttpResponse)

			go func() {
				page := r.URL.Query().Get("page")

				if page == "" {
					page = "1"
				}

				pageNum, err := strconv.ParseUint(page, 10, 32)

				if err != nil {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				limit := perPage
				offset := (pageNum - 1) * perPage

				rows, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				var bots []*types.IndexBot

				err = pgxscan.ScanAll(&bots, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
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

				err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots").Scan(&count)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
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

				resp <- types.HttpResponse{
					Json: data,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:  "GET",
			Path:    "/bots/{id}",
			OpId:    "get_bot",
			Summary: "Get Bot",
			Description: `
Gets a bot by id or name

**Some things to note:**

-` + constants.BackTick + constants.BackTick + `external_source` + constants.BackTick + constants.BackTick + ` shows the source of where a bot came from (Metro Reviews etc etr.). If this is set to ` + constants.BackTick + constants.BackTick + `metro` + constants.BackTick + constants.BackTick + `, then ` + constants.BackTick + constants.BackTick + `list_source` + constants.BackTick + constants.BackTick + ` will be set to the metro list ID where it came from` + `
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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				name := chi.URLParam(r, "id")

				name = strings.ToLower(name)

				if name == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Check cache, this is how we can avoid hefty ratelimits
				cache := state.Redis.Get(ctx, "bc-"+name).Val()
				if cache != "" {
					resp <- types.HttpResponse{
						Data: cache,
						Headers: map[string]string{
							"X-Popplio-Cached": "true",
						},
					}
					return
				}

				var bot types.Bot

				var err error

				row, err := state.Pool.Query(ctx, "SELECT "+botCols+" FROM bots WHERE (bot_id = $1 OR vanity = $1) LIMIT 1", name)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				err = pgxscan.ScanOne(&bot, row)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				err = utils.ParseBot(ctx, state.Pool, &bot, state.Discord, state.Redis)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				var uniqueClicks int64
				err = state.Pool.QueryRow(ctx, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				bot.UniqueClicks = uniqueClicks

				/* Removing or modifying fields directly in API is very dangerous as scrapers will
				 * just ignore owner checks anyways or cross-reference via another list. Also we
				 * want to respect the permissions of the owner if they're the one giving permission,
				 * blocking IPs is a better idea to this
				 */

				resp <- types.HttpResponse{
					Json:      bot,
					CacheKey:  "bc-" + name,
					CacheTime: time.Minute * 3,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:  "POST",
			Path:    "/bots/stats",
			OpId:    "post_stats",
			Summary: "Post Bot Stats",
			Description: `
	This endpoint can be used to post the stats of a bot.
	
	The variation` + constants.BackTick + `/bots/{bot_id}/stats` + constants.BackTick + ` can also be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**
	
	**Example:**
	
	` + constants.BackTick + constants.BackTick + constants.BackTick + `py
	import requests
	
	req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})
	
	print(req.json())
	` + constants.BackTick + constants.BackTick + constants.BackTick + "\n\n",
			Tags:     []string{tagName},
			Req:      types.BotStatsDocs{},
			Resp:     types.ApiError{},
			AuthType: []string{"Bot"},
		})
		r.Post("/stats", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				if r.Body == nil {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				var id *string

				// Check token
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					id = utils.AuthCheck(r.Header.Get("Authorization"), true)

					if id == nil {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				defer r.Body.Close()

				var payload types.BotStats

				bodyBytes, err := io.ReadAll(r.Body)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				err = json.Unmarshal(bodyBytes, &payload)

				if err != nil {
					if r.URL.Query().Get("count") != "" {
						payload = types.BotStats{}
					} else {
						state.Logger.Error(err)
						resp <- types.HttpResponse{
							Data:   constants.BadRequestStats,
							Status: http.StatusBadRequest,
						}
						return
					}
				}

				if r.URL.Query().Get("count") != "" {
					count, err := strconv.ParseUint(r.URL.Query().Get("count"), 10, 32)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
						return
					}

					var countAny any = count

					payload.Count = &countAny
				}

				servers, shards, users := payload.GetStats()

				if servers > 0 {
					_, err = state.Pool.Exec(ctx, "UPDATE bots SET servers = $1 WHERE bot_id = $2", servers, id)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
						return
					}
				}

				if shards > 0 {
					_, err = state.Pool.Exec(ctx, "UPDATE bots SET shards = $1 WHERE bot_id = $2", shards, id)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
						return
					}
				}

				if users > 0 {
					_, err = state.Pool.Exec(ctx, "UPDATE bots SET users = $1 WHERE bot_id = $2", users, id)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
						return
					}
				}

				// Get name and vanity, delete from cache
				var vanity string

				state.Pool.QueryRow(ctx, "SELECT vanity FROM bots WHERE bot_id = $1", id).Scan(&vanity)

				// Delete from cache
				state.Redis.Del(ctx, "bc-"+vanity)
				state.Redis.Del(ctx, "bc-"+*id)

				resp <- types.HttpResponse{
					Data: constants.Success,
				}
			}()

			utils.Respond(ctx, w, resp)
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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				name := chi.URLParam(r, "id")

				if name == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				cache := state.Redis.Get(ctx, "seob:"+name).Val()
				if cache != "" {
					resp <- types.HttpResponse{
						Data: cache,
						Headers: map[string]string{
							"X-Popplio-Cached": "true",
						},
					}
					return
				}

				var botId string
				var short string
				err := state.Pool.QueryRow(ctx, "SELECT bot_id, short FROM bots WHERE (bot_id = $1 OR vanity = $1)", name).Scan(&botId, &short)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				bot, err := utils.GetDiscordUser(botId)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				seoData := types.SEO{
					ID:       bot.ID,
					Username: bot.Username,
					Avatar:   bot.Avatar,
					Short:    short,
				}

				resp <- types.HttpResponse{
					Json:      seoData,
					CacheKey:  "seob:" + name,
					CacheTime: 30 * time.Minute,
				}
			}()

			utils.Respond(ctx, w, resp)
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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				rows, err := state.Pool.Query(ctx, "SELECT "+reviewCols+" FROM reviews WHERE (bot_id = $1 OR vanity = $1)", chi.URLParam(r, "id"))

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				var reviews []types.Review = []types.Review{}

				err = pgxscan.ScanAll(&reviews, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				var allReviews = types.ReviewList{
					Reviews: reviews,
				}

				resp <- types.HttpResponse{
					Json: allReviews,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}
