package get_user_seo

import (
	"net/http"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}/seo",
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
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	if name == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	cache := state.Redis.Get(d.Context, "seou:"+name).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var about string
	var userId string
	err := state.Pool.QueryRow(d.Context, "SELECT about, user_id FROM users WHERE user_id = $1 OR username = $1", name).Scan(&about, &userId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	user, err := utils.GetDiscordUser(userId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	seo := types.SEO{
		ID:       user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
		Short:    about,
	}

	return api.HttpResponse{
		Json:      seo,
		CacheKey:  "seou:" + name,
		CacheTime: 30 * time.Minute,
	}
}
