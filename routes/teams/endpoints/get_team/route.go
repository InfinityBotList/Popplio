package get_team

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := chi.URLParam(r, "id")

	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", id).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var name string
	var ownerId string
	var avatar string

	err = state.Pool.QueryRow(d.Context, "SELECT name, owner, avatar FROM teams WHERE id = $1", id).Scan(&name, &ownerId, &avatar)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	owner, err := utils.GetDiscordUser(d.Context, ownerId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Next handle members
	var members = []types.TeamMember{}

	rows, err := state.Pool.Query(d.Context, "SELECT user_id, perms FROM team_members WHERE team_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	for rows.Next() {
		var userId string
		var perms []string

		err = rows.Scan(&userId, &perms)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		user, err := utils.GetDiscordUser(d.Context, userId)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		members = append(members, types.TeamMember{
			User:  user,
			Perms: perms,
		})
	}

	return api.HttpResponse{
		Json: types.Team{
			ID:        id,
			Name:      name,
			MainOwner: owner,
			Avatar:    avatar,
			Members:   members,
		},
	}
}
