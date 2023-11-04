package get_user_seo

import (
	"net/http"
	"time"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User SEO Info",
		Description: "Gets a users SEO data by id",
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	name := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seou:"+name).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var about string
	var userId string
	err := state.Pool.QueryRow(d.Context, "SELECT about, user_id FROM users WHERE user_id = $1", name).Scan(&about, &userId)

	if err != nil {
		state.Logger.Error("Failed to get user seo", zap.Error(err), zap.String("user_id", name))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	user, err := dovewing.GetUser(d.Context, userId, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to get user seo", zap.Error(err), zap.String("user_id", name))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	seo := types.SEO{
		ID:     user.ID,
		Name:   user.DisplayName,
		Avatar: user.Avatar,
		Short:  about,
	}

	return uapi.HttpResponse{
		Json:      seo,
		CacheKey:  "seou:" + name,
		CacheTime: 30 * time.Minute,
	}
}
