// Package get_pack_seo implements GET /packs/{id}/seo — "Get Pack SEO Info".
//
// Gets the minimal SEO information about a pack for embed/search purposes.
// Used by v4 website for meta tags
package get_pack_seo

import (
	"errors"
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Pack SEO Info",
		Description: "Gets the minimal SEO information about a pack for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The packs ID, name or vanity",
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
	var packName string
	err := state.Pool.QueryRow(d.Context, "SELECT name, short FROM packs WHERE url = $1", id).Scan(&packName, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Failed to get pack seo", err, zap.String("url", id))
	}

	seoData := types.SEO{
		ID:     id,
		Name:   packName,
		Avatar: "",
		Short:  short,
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
