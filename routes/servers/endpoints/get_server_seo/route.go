// Package get_server_seo implements GET /servers/{id}/seo — "Get Server SEO
// Info".
//
// Gets the minimal SEO information about a server for embed/search purposes.
// Used by v4 website for meta tags
package get_server_seo

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
		Summary:     "Get Server SEO Info",
		Description: "Gets the minimal SEO information about a server for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	var name, short string
	err := state.Pool.QueryRow(d.Context, "SELECT name, short FROM servers WHERE server_id = $1", id).Scan(&name, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting server [queryrow]", err, zap.String("serverID", id))
	}

	seoData := types.SEO{
		ID:    id,
		Name:  name,
		Short: short,
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
