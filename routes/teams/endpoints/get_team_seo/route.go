package get_team_seo

import (
	"net/http"
	"time"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team SEO Info",
		Description: "Gets the minimal SEO information about a team for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The team ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	tid := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seot:"+tid).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	// Convert ID to UUID
	if !utils.IsValidUUID(tid) {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var id string
	var name string
	var avatar string
	err := state.Pool.QueryRow(d.Context, "SELECT id, name, avatar FROM teams WHERE id = $1", tid).Scan(&id, &name, &avatar)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	seoData := types.SEO{
		ID:       id,
		Username: name,
		Avatar:   avatar,
		Short:    "View the team " + name + " on Infinity Bot List",
	}

	return api.HttpResponse{
		Json:      seoData,
		CacheKey:  "seot:" + name,
		CacheTime: 2 * time.Minute,
	}
}
