package reset_user_token

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/eureka/crypto"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Reset User Token",
		Description: "Resets the token of a user. The user should be logged out after this. Returns 204 on success.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	utils.ClearUserCache(d.Context, d.Auth.ID)

	token := crypto.RandString(128)

	_, err := state.Pool.Exec(d.Context, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, d.Auth.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}