package get_user_perms

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
		Summary:     "Get User Perms",
		Description: "Gets a users permissions by ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserPerm{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := chi.URLParam(r, "id")

	var experiments []string
	var staff bool
	var admin bool
	var hadmin bool
	var ibldev bool
	var iblhdev bool
	err := state.Pool.QueryRow(d.Context, "SELECT experiments, staff, admin, hadmin, ibldev, iblhdev FROM users WHERE user_id = $1", id).Scan(&experiments, &staff, &admin, &hadmin, &ibldev, &iblhdev)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	user, err := utils.GetDiscordUser(d.Context, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	up := types.UserPerm{
		ID:          id,
		User:        user,
		Experiments: experiments,
		Staff:       staff,
		Admin:       admin,
		HAdmin:      hadmin,
		IBLDev:      ibldev,
		IBLHDev:     iblhdev,
	}

	return api.HttpResponse{
		Json: up,
	}
}
