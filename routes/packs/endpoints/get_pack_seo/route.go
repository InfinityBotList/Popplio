package get_pack_seo

import (
	"net/http"
	"time"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Pack SEO Info",
		Description: "Gets the minimal SEO information about a pack for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The packs ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seop:"+id).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", id).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var short string
	var packName string
	err = state.Pool.QueryRow(d.Context, "SELECT name, short FROM packs WHERE url = $1", id).Scan(&packName, &short)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	seoData := types.SEO{
		ID:             id,
		Name:           packName,
		UsernameLegacy: packName,
		Avatar:         "",
		Short:          short,
	}

	return uapi.HttpResponse{
		Json:      seoData,
		CacheKey:  "seop:" + id,
		CacheTime: 30 * time.Minute,
	}
}
