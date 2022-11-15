package list

import (
	"io"
	"net/http"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"
	"strings"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
)

const tagName = "List Stats"

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary

	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	indexPackColsArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are basic statistics of our list."
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/list", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/list/index",
			OpId:        "get_list_index",
			Summary:     "Get List Index",
			Description: "Gets the index of the list. Returns a ``Index`` object",
			Tags:        []string{tagName},
			Resp:        types.ListIndex{},
		})
		r.Get("/index", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				// Check cache, this is how we can avoid hefty ratelimits
				cache := state.Redis.Get(ctx, "indexcache").Val()
				if cache != "" {
					resp <- types.HttpResponse{
						Data: cache,
						Headers: map[string]string{
							"X-Popplio-Cached": "true",
						},
					}
					return
				}

				listIndex := types.ListIndex{}

				certRow, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE certified = true AND type = 'approved' ORDER BY votes DESC LIMIT 9")
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				certDat := []types.IndexBot{}
				err = pgxscan.ScanAll(&certDat, certRow)
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				listIndex.Certified = certDat

				mostViewedRow, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY clicks DESC LIMIT 9")
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				mostViewedDat := []types.IndexBot{}
				err = pgxscan.ScanAll(&mostViewedDat, mostViewedRow)
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				listIndex.MostViewed = mostViewedDat

				recentlyAddedRow, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY date DESC LIMIT 9")
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				recentlyAddedDat := []types.IndexBot{}
				err = pgxscan.ScanAll(&recentlyAddedDat, recentlyAddedRow)
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				listIndex.RecentlyAdded = recentlyAddedDat

				topVotedRow, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE type = 'approved' ORDER BY votes DESC LIMIT 9")
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				topVotedDat := []types.IndexBot{}
				err = pgxscan.ScanAll(&topVotedDat, topVotedRow)
				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}
				listIndex.TopVoted = topVotedDat

				rows, err := state.Pool.Query(ctx, "SELECT "+indexPackCols+" FROM packs ORDER BY date DESC")

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				var packs []*types.IndexBotPack

				err = pgxscan.ScanAll(&packs, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				listIndex.Packs = packs

				resp <- types.HttpResponse{
					Json:      listIndex,
					CacheKey:  "indexcache",
					CacheTime: 15 * time.Minute,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/list/stats",
			OpId:        "get_list_stats",
			Summary:     "Get List Statistics",
			Description: "Gets the statistics of the list",
			Tags:        []string{tagName},
			Resp: types.ListStats{
				Bots: []types.ListStatsBot{},
			},
		})
		r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				listStats := types.ListStats{}

				bots, err := state.Pool.Query(ctx, "SELECT bot_id, name, short, type, owner, additional_owners, avatar, certified, claimed FROM bots")

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				defer bots.Close()

				for bots.Next() {
					var botId string
					var name string
					var short string
					var typeStr string
					var owner string
					var additionalOwners []string
					var avatar string
					var certified bool
					var claimed bool

					err := bots.Scan(&botId, &name, &short, &typeStr, &owner, &additionalOwners, &avatar, &certified, &claimed)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
						return
					}

					listStats.Bots = append(listStats.Bots, types.ListStatsBot{
						BotID:              botId,
						Name:               name,
						Short:              short,
						Type:               typeStr,
						AvatarDB:           avatar,
						MainOwnerID:        owner,
						AdditionalOwnerIDS: additionalOwners,
						Certified:          certified,
						Claimed:            claimed,
					})
				}

				var activeStaff int64
				err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE staff = true").Scan(&activeStaff)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				listStats.TotalStaff = activeStaff

				var totalUsers int64
				err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				listStats.TotalUsers = totalUsers

				var totalVotes int64
				err = state.Pool.QueryRow(ctx, "SELECT SUM(votes) FROM bots").Scan(&totalVotes)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				listStats.TotalVotes = totalVotes

				var totalPacks int64
				err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM packs").Scan(&totalPacks)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				listStats.TotalPacks = totalPacks

				var totalTickets int64
				err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM transcripts").Scan(&totalTickets)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				listStats.TotalTickets = totalTickets

				resp <- types.HttpResponse{
					Json: listStats,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/list/vote-info",
			OpId:        "get_vote_info",
			Summary:     "Get Vote Info",
			Description: "Returns basic voting info such as if its a weekend double vote.",
			Resp:        types.VoteInfo{Weekend: true},
			Tags:        []string{tagName},
		})
		r.Get("/vote-info", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var payload = types.VoteInfo{
					Weekend: utils.GetDoubleVote(),
				}

				resp <- types.HttpResponse{
					Json: payload,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "POST",
			Path:        "/list/webhook-test",
			OpId:        "webhook_test",
			Summary:     "Test Webhook",
			Description: "Sends a test webhook to allow testing your vote system. **All fields are mandatory for this endpoint**",
			Req:         types.WebhookPost{},
			Resp:        types.ApiError{},
			Tags:        []string{tagName},
		})
		r.Post("/webhook-test", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				defer r.Body.Close()

				var payload types.WebhookPost

				bodyBytes, err := io.ReadAll(r.Body)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				err = json.Unmarshal(bodyBytes, &payload)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if utils.IsNone(payload.URL) && utils.IsNone(payload.URL2) {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				payload.Test = true // Always true

				var err1 error

				if !utils.IsNone(payload.URL) {
					err1 = webhooks.Send(payload)
				}

				var err2 error

				if !utils.IsNone(payload.URL2) {
					payload.URL = payload.URL2 // Test second enpdoint if it's not empty
					err2 = webhooks.Send(payload)
				}

				var errD = types.ApiError{}

				if err1 != nil {
					state.Logger.Error(err1)

					errD.Message = err1.Error()
					errD.Error = true
				}

				if err2 != nil {
					state.Logger.Error(err2)

					errD.Message += "|" + err2.Error()
					errD.Error = true
				}

				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   errD,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}
