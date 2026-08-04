// Package get_blog_list implements GET /blogs/@all — "Get Blog List".
//
// Gets all blog posts on the list in condensed form
package get_blog_list

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	blogColsArr = db.GetCols(types.BlogListPost{})

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
	rows, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs WHERE draft = false ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Error while fetching blog posts [db query]", err)
	}

	blogPosts, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.BlogListPost])

	if err != nil {
		return resp.Err("Error while fetching blog posts [collect]", err)
	}

	for i := range blogPosts {
		blogPosts[i].Author, err = dovewing.GetUser(d.Context, blogPosts[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error while getting user [dovewing]", err, zap.String("user_id", blogPosts[i].UserID))
		}
	}

	return uapi.HttpResponse{
		Json: types.Blog{
			Posts: blogPosts,
		},
	}
}
