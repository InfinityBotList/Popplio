package packs

import (
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

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Bot Packs"
	perPage = 12
)

var (
	packColArr = utils.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")

	indexPackColArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols   = strings.Join(indexPackColArr, ",")
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to IBL packs"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/packs", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/packs/{id}",
			OpId:        "get_packs",
			Summary:     "Get Packs",
			Description: "Gets a pack on the list based on either URL or Name.",
			Tags:        []string{tagName},
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "The ID of the pack.",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.BotPack{},
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				var pack types.BotPack

				row, err := state.Pool.Query(ctx, "SELECT "+packCols+" FROM packs WHERE url = $1 OR name = $1", id)

				if err != nil {
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				err = pgxscan.ScanOne(&pack, row)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				err = utils.ResolveBotPack(ctx, state.Pool, &pack)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				resp <- types.HttpResponse{
					Json: pack,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/packs/all",
			OpId:        "get_all_packs",
			Summary:     "Get All Packs",
			Description: "Gets all packs on the list. Returns a ``Index`` object",
			Tags:        []string{tagName},
			Resp:        types.AllPacks{},
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

				// Check cache, this is how we can avoid hefty ratelimits
				cache := state.Redis.Get(ctx, "pca-"+strconv.FormatUint(pageNum, 10)).Val()
				if cache != "" {
					resp <- types.HttpResponse{
						Data: cache,
						Headers: map[string]string{
							"X-Popplio-Cached": "true",
						},
					}
					return
				}

				limit := perPage
				offset := (pageNum - 1) * perPage

				rows, err := state.Pool.Query(ctx, "SELECT "+indexPackCols+" FROM packs ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				packs := []types.IndexBotPack{}

				err = pgxscan.ScanAll(&packs, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				for _, pack := range packs {
					pack.Votes, err = utils.ResolvePackVotes(ctx, pack.URL)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
						return
					}
				}

				var previous strings.Builder

				// More optimized string concat
				previous.WriteString(os.Getenv("SITE_URL"))
				previous.WriteString("/packs/all?page=")
				previous.WriteString(strconv.FormatUint(pageNum-1, 10))

				if pageNum-1 < 1 || pageNum == 0 {
					previous.Reset()
				}

				var count uint64

				err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM packs").Scan(&count)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				var next strings.Builder

				next.WriteString(os.Getenv("SITE_URL"))
				next.WriteString("/packs/all?page=")
				next.WriteString(strconv.FormatUint(pageNum+1, 10))

				if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
					next.Reset()
				}

				data := types.AllPacks{
					Count:    count,
					Results:  packs,
					PerPage:  perPage,
					Previous: previous.String(),
					Next:     next.String(),
				}

				resp <- types.HttpResponse{
					Json:      data,
					CacheKey:  "pca-" + strconv.FormatUint(pageNum, 10),
					CacheTime: 2 * time.Minute,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}
