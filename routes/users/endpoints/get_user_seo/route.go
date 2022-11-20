package get_user_seo

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}/seo",
		OpId:        "get_user_seo",
		Summary:     "Get User SEO Info",
		Description: "Gets a users SEO data by id or username",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.SEO{},
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	name := chi.URLParam(r, "id")

	if name == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	cache := state.Redis.Get(d.Context, "seou:"+name).Val()
	if cache != "" {
		d.Resp <- types.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	var about string
	var userId string
	err := state.Pool.QueryRow(d.Context, "SELECT about, user_id FROM users WHERE user_id = $1 OR username = $1", name).Scan(&about, &userId)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	user, err := utils.GetDiscordUser(userId)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	seo := types.SEO{
		ID:       user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
		Short:    about,
	}

	d.Resp <- types.HttpResponse{
		Json:      seo,
		CacheKey:  "seou:" + name,
		CacheTime: 30 * time.Minute,
	}
}
