package get_blog

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	blogColsArr = utils.GetCols(types.Blog{})

	blogCols = strings.Join(blogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog",
		Description: "Gets all blog posts on the list. Returns a ``Blog`` object",
		Resp:        types.Blog{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var blogPosts []types.BlogPost

	rows, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&blogPosts, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if len(blogPosts) == 0 {
		blogPosts = []types.BlogPost{}
	}

	return api.HttpResponse{
		Json: types.Blog{
			Posts: blogPosts,
		},
	}
}
