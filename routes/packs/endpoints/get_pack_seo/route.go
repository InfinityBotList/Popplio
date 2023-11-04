package get_pack_seo

import (
	"errors"
	"net/http"
	"time"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

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

	var short string
	var packName string
	err := state.Pool.QueryRow(d.Context, "SELECT name, short FROM packs WHERE url = $1", id).Scan(&packName, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Failed to get pack seo", zap.Error(err), zap.String("url", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	seoData := types.SEO{
		ID:     id,
		Name:   packName,
		Avatar: "",
		Short:  short,
	}

	return uapi.HttpResponse{
		Json:      seoData,
		CacheKey:  "seop:" + id,
		CacheTime: 30 * time.Minute,
	}
}
