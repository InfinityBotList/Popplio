package get_blog_list

import (
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var blogPosts []types.BlogListPost

	err = pgxscan.ScanAll(&blogPosts, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range blogPosts {
		blogPosts[i].Author, err = dovewing.GetDiscordUser(d.Context, blogPosts[i].UserID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return uapi.HttpResponse{
		Json: types.Blog{
			Posts: blogPosts,
		},
	}
}
