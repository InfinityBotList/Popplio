package get_user

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	userColsArr = utils.GetCols(types.User{})
	userCols    = strings.Join(userColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}",
		OpId:        "get_user",
		Summary:     "Get User",
		Description: "Gets a user by id or username",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.User{},
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	name := chi.URLParam(r, "id")

	if name == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	if name == "undefined" {
		d.Resp <- types.HttpResponse{
			Status: http.StatusOK,
			Data:   `{"error":"false","message":"Handling known issue"}`,
		}
		return
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "uc-"+name).Val()
	if cache != "" {
		d.Resp <- types.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	var user types.User

	var err error

	row, err := state.Pool.Query(d.Context, "SELECT "+userCols+" FROM users WHERE user_id = $1 OR username = $1", name)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	err = pgxscan.ScanOne(&user, row)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	err = utils.ParseUser(d.Context, state.Pool, &user, state.Discord, state.Redis)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	/* Removing or modifying fields directly in API is very dangerous as scrapers will
	 * just ignore owner checks anyways or cross-reference via another list. Also we
	 * want to respect the permissions of the owner if they're the one giving permission,
	 * blocking IPs is a better idea to this
	 */

	d.Resp <- types.HttpResponse{
		Json:      user,
		CacheKey:  "uc-" + name,
		CacheTime: 3 * time.Minute,
	}
}
