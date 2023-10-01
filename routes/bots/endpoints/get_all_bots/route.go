package get_all_bots

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/assets"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const perPage = 12

var (
	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list. Returns a set of paginated ``IndexBot`` objects",
		Resp:        types.PagedResult[[]types.IndexBot]{},
		RespName:    "PagedResultIndexBot",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "allbots-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	var rows pgx.Rows

	rows, err = state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Set the user for each bot
	for i := range bots {
		botUser, err := dovewing.GetUser(d.Context, bots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].User = botUser

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", bots[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].Vanity = code
		bots[i].Banner = assets.BannerInfo("bots", bots[i].BotID)
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[[]types.IndexBot]{
		Count:   count,
		Results: bots,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json:      data,
		CacheKey:  "allbots-" + strconv.FormatUint(pageNum, 10),
		CacheTime: 10 * time.Minute,
	}
}
