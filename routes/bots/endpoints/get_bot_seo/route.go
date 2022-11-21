package get_bot_seo

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/bots/{id}/seo",
		OpId:        "get_bot_seo",
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
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		d.Resp <- api.DefaultResponse(http.StatusBadRequest)
		return
	}

	cache := state.Redis.Get(d.Context, "seob:"+name).Val()
	if cache != "" {
		d.Resp <- api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
		return
	}

	var botId string
	var short string
	err := state.Pool.QueryRow(d.Context, "SELECT bot_id, short FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", name).Scan(&botId, &short)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	bot, err := utils.GetDiscordUser(botId)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	seoData := types.SEO{
		ID:       bot.ID,
		Username: bot.Username,
		Avatar:   bot.Avatar,
		Short:    short,
	}

	d.Resp <- api.HttpResponse{
		Json:      seoData,
		CacheKey:  "seob:" + name,
		CacheTime: 30 * time.Minute,
	}
}
