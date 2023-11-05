package get_server_seo

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
		Summary:     "Get Server SEO Info",
		Description: "Gets the minimal SEO information about a server for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seos:"+id).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var name, avatar, short string
	err := state.Pool.QueryRow(d.Context, "SELECT name, avatar, short FROM servers WHERE server_id = $1", id).Scan(&name, &avatar, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Error while getting server [queryrow]", zap.Error(err), zap.String("serverID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	seoData := types.SEO{
		ID:     id,
		Name:   name,
		Avatar: avatar,
		Short:  short,
	}

	return uapi.HttpResponse{
		Json:      seoData,
		CacheKey:  "seos:" + id,
		CacheTime: 30 * time.Minute,
	}
}
