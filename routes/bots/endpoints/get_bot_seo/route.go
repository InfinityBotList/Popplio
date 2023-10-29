package get_bot_seo

import (
	"errors"
	"net/http"
	"time"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot SEO Info",
		Description: "Gets the minimal SEO information about a bot for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seob:"+id).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var short string
	err := state.Pool.QueryRow(d.Context, "SELECT short FROM bots WHERE bot_id = $1", id).Scan(&short)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Error while getting bot [queryrow]", zap.Error(err), zap.String("botID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	bot, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Error while getting bot user [dovewing]", zap.Error(err), zap.String("botID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	seoData := types.SEO{
		ID:     bot.ID,
		Name:   bot.DisplayName,
		Avatar: bot.Avatar,
		Short:  short,
	}

	return uapi.HttpResponse{
		Json:      seoData,
		CacheKey:  "seob:" + id,
		CacheTime: 30 * time.Minute,
	}
}
