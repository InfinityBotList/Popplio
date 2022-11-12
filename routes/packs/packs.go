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

	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

const tagName = "Bot Packs"

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary

	packColArr = utils.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")
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
			var id = chi.URLParam(r, "id")

			if id == "" {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			var pack types.BotPack

			row, err := state.Pool.Query(state.Context, "SELECT "+packCols+" FROM packs WHERE url = $1 OR name = $1", id)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			err = pgxscan.ScanOne(&pack, row)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			err = utils.ResolveBotPack(state.Context, state.Pool, &pack, state.Discord, state.Redis)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			bytes, err := json.Marshal(pack)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/packs/all",
			OpId:        "get_all_packs",
			Summary:     "Get All Packs",
			Description: "Gets all packs on the list.",
			Tags:        []string{tagName},
			Resp:        types.AllPacks{},
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

			// Check cache, this is how we can avoid hefty ratelimits
			cache := state.Redis.Get(state.Context, "pca-"+strconv.FormatUint(pageNum, 10)).Val()
			if cache != "" {
				w.Header().Add("X-Popplio-Cached", "true")
				w.Write([]byte(cache))
				return
			}

			limit := perPage
			offset := (pageNum - 1) * perPage

			rows, err := state.Pool.Query(state.Context, "SELECT "+packCols+" FROM packs ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var packs []*types.BotPack

			err = pgxscan.ScanAll(&packs, rows)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			for _, pack := range packs {
				err := utils.ResolveBotPack(state.Context, state.Pool, pack, state.Discord, state.Redis)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
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

			err = state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM packs").Scan(&count)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
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

			bytes, err := json.Marshal(data)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			state.Redis.Set(state.Context, "pca-"+strconv.FormatUint(pageNum, 10), bytes, 2*time.Minute)

			w.Write(bytes)
		})
	})
}
