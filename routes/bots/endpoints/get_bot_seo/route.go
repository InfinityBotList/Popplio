package get_bot_seo

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
		Path:        "/bots/{id}/seo",
		Summary:     "Get Bot SEO Info",
		Description: "Gets the minimal SEO information about a bot for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seob:"+name).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var short string
	err = state.Pool.QueryRow(d.Context, "SELECT short FROM bots WHERE bot_id = $1", id).Scan(&short)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	bot, err := utils.GetDiscordUser(id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	seoData := types.SEO{
		ID:       bot.ID,
		Username: bot.Username,
		Avatar:   bot.Avatar,
		Short:    short,
	}

	return api.HttpResponse{
		Json:      seoData,
		CacheKey:  "seob:" + name,
		CacheTime: 30 * time.Minute,
	}
}
