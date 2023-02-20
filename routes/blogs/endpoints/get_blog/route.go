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
	blogColsArr = utils.GetCols(types.BlogPost{})

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
	rows, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var blogPosts []types.BlogPost

	err = pgxscan.ScanAll(&blogPosts, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: types.Blog{
			Posts: blogPosts,
		},
	}
}
