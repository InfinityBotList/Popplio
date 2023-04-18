package get_team

import (
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team",
		Description: "Gets a team by ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Team{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	// Convert ID to UUID
	if !utils.IsValidUUID(id) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", id).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	team, err := utils.ResolveTeam(d.Context, id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: team,
	}
}
