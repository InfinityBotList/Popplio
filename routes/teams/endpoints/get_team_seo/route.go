package get_team_seo

import (
	"net/http"
	"strconv"
	"time"

	"popplio/assetmanager"
	"popplio/state"
	"popplio/types"

	"github.com/google/uuid"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team SEO Info",
		Description: "Gets the minimal SEO information about a team for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The team ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	tid := chi.URLParam(r, "id")

	cache := state.Redis.Get(d.Context, "seot:"+tid).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	// Convert ID to UUID
	if _, err := uuid.Parse(tid); err != nil {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var id string
	var name string
	var short string
	err := state.Pool.QueryRow(d.Context, "SELECT id, name, short FROM teams WHERE id = $1", tid).Scan(&id, &name, &short)

	if err != nil {
		state.Logger.Error("Error getting team SEO info [db queryrow]", zap.Error(err), zap.String("id", tid))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	avatar := assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, id)

	var avatarPath string

	if avatar.Exists {
		avatarPath = state.Config.Sites.CDN + "/" + avatar.Path + "?ts=" + strconv.FormatInt(avatar.LastModified.Unix(), 10)
	} else {
		avatarPath = state.Config.Sites.CDN + "/" + avatar.DefaultPath
	}

	seoData := types.SEO{
		ID:     id,
		Name:   name,
		Avatar: avatarPath,
		Short: func() string {
			if short == "" {
				return "View the team " + name + " on Infinity List"
			}

			return short
		}(),
	}

	return uapi.HttpResponse{
		Json:      seoData,
		CacheKey:  "seot:" + name,
		CacheTime: 2 * time.Minute,
	}
}
