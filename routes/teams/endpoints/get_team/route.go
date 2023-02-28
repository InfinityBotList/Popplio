package get_team

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"time"

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

	// Convert ID to UUID
	if !utils.IsValidUUID(id) {
		return api.DefaultResponse(http.StatusNotFound)
	}

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
	var avatar string

	err = state.Pool.QueryRow(d.Context, "SELECT name, avatar FROM teams WHERE id = $1", id).Scan(&name, &avatar)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Next handle members
	var members = []types.TeamMember{}

	rows, err := state.Pool.Query(d.Context, "SELECT user_id, perms, created_at FROM team_members WHERE team_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	for rows.Next() {
		var userId string
		var perms []teams.TeamPermission
		var createdAt time.Time

		err = rows.Scan(&userId, &perms, &createdAt)

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
			User:      user,
			Perms:     teams.NewPermissionManager(perms).Perms(),
			CreatedAt: createdAt,
		})
	}

	// Bots
	bots, err := utils.ResolveTeamBots(d.Context, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: types.Team{
			ID:       id,
			Name:     name,
			Avatar:   avatar,
			Members:  members,
			UserBots: bots,
		},
	}
}
