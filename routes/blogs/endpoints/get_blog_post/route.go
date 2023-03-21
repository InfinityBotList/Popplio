package get_blog_post

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	blogColsArr = utils.GetCols(types.BlogPost{})

	blogCols = strings.Join(blogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog Post",
		Description: "Gets a blog posts on the list",
		Resp:        types.BlogPost{},
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM blogs WHERE slug = $1", chi.URLParam(r, "slug")).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs WHERE slug = $1", chi.URLParam(r, "slug"))

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var blogPost types.BlogPost

	err = pgxscan.ScanOne(&blogPost, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	blogPost.Author, err = dovewing.GetDiscordUser(d.Context, blogPost.UserID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: blogPost,
	}
}
