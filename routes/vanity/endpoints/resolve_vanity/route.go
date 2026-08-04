// Package resolve_vanity implements GET /vanity/{code} — "Resolve Vanity".
//
// Resolve a vanity by its code or (then) its target id
package resolve_vanity

import (
	"net/http"
	"popplio/api/resp"

	"popplio/routes/vanity/assets"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Resolve Vanity",
		Description: "Resolve a vanity by its code or (then) its target id",
		Resp:        types.Vanity{},
		Params: []docs.Parameter{
			{
				Name:        "code",
				In:          "path",
				Description: "The vanity code",
				Required:    true,
				Schema:      docs.IdSchema,
			},
			{
				Name:        "itag",
				In:          "query",
				Description: "Resolve based on itag",
				Required:    false,
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Fetch the vanity, first attempt code
	code := chi.URLParam(r, "code")
	itag := r.URL.Query().Get("itag")

	var v *types.Vanity
	var err error

	if itag == "true" {
		v, err = assets.ResolveVanityByItag(d.Context, code)
	} else {
		v, err = assets.ResolveVanity(d.Context, code)
	}

	if err != nil {
		return resp.ErrBody("Failed to resolve vanity", "An internal error occurred", err, zap.String("code", code))
	}

	if v == nil {
		return resp.NotFound("This entity does not exist")
	}

	return uapi.HttpResponse{
		Json: v,
	}
}
