package get_blog_list

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/doclib"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	blogColsArr = utils.GetCols(types.BlogListPost{})

	blogCols = strings.Join(blogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog List",
		Description: "Gets all blog posts on the list in condensed form",
		Resp:        types.Blog{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var blogPosts []types.BlogListPost

	err = pgxscan.ScanAll(&blogPosts, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range blogPosts {
		blogPosts[i].Author, err = utils.GetDiscordUser(d.Context, blogPosts[i].UserID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.HttpResponse{
		Json: types.Blog{
			Posts: blogPosts,
		},
	}
}
