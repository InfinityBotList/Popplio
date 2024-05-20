package get_team_seo

import (
	"net/http"

	"popplio/assetmanager"
	"popplio/state"
	"popplio/types"

	"github.com/google/uuid"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
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

	// Convert ID to UUID
	if _, err := uuid.Parse(tid); err != nil {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var id string
	var name string
	var short pgtype.Text
	err := state.Pool.QueryRow(d.Context, "SELECT id, name, short FROM teams WHERE id = $1", tid).Scan(&id, &name, &short)

	if err != nil {
		state.Logger.Error("Error getting team SEO info [db queryrow]", zap.Error(err), zap.String("id", tid))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	avatar := assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, id)

	seoData := types.SEO{
		ID:     id,
		Name:   name,
		Avatar: assetmanager.ResolveAssetMetadataToUrl(avatar),
		Short: func() string {
			if !short.Valid || short.String == "" {
				return "View the team " + name + " on Infinity List"
			}

			return short.String
		}(),
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
