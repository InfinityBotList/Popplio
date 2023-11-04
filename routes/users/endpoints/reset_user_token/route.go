package reset_user_token

import (
	"net/http"
	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	token := crypto.RandString(128)

	_, err := state.Pool.Exec(d.Context, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, d.Auth.ID)

	if err != nil {
		state.Logger.Error("Failed to reset user token", zap.Error(err), zap.String("user_id", d.Auth.ID), zap.String("token", token))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
