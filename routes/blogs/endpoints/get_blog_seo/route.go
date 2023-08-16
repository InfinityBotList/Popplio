package get_blog_seo

import (
	"net/http"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog Post",
		Description: "Gets the minimal SEO information about a blogpost for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "slug",
				Description: "The slug of the blog post",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	slug := chi.URLParam(r, "slug")

	cache := state.Redis.Get(d.Context, "seoblog:"+slug).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var title string
	var description string

	err := state.Pool.QueryRow(d.Context, "SELECT title, description FROM blogs WHERE slug = $1", slug).Scan(&title, &description)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	seo := types.SEO{
		ID:     slug,
		Name:   title,
		Avatar: "",
		Short:  description,
	}

	return uapi.HttpResponse{
		Json:      seo,
		CacheKey:  "seoblog:" + slug,
		CacheTime: 30 * time.Minute,
	}
}
