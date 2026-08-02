// Package get_bot_seo implements GET /bots/{id}/seo — "Get Bot SEO Info".
//
// Gets the minimal SEO information about a bot for embed/search purposes.
// Used by v4 website for meta tags
package get_bot_seo

import (
	"errors"
	"net/http"
	"popplio/api/resp"

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

	var short string
	err := state.Pool.QueryRow(d.Context, "SELECT short FROM bots WHERE bot_id = $1", id).Scan(&short)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting bot [queryrow]", err, zap.String("botID", id))
	}

	bot, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error while getting bot user [dovewing]", err, zap.String("botID", id))
	}

	seoData := types.SEO{
		ID:     bot.ID,
		Name:   bot.DisplayName,
		Avatar: bot.Avatar,
		Short:  short,
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
