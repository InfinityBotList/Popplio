package check_booster_status

import (
	"net/http"

	"popplio/routes/payments/assets"
	"popplio/types"

	"github.com/disgoorg/snowflake/v2"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Check Booster Status",
		Description: "Returns the booster status of a user. This can be used to check eligibility to redeem booster perks.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BoosterStatus{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	idSnow, err := snowflake.Parse(id)

	if err != nil {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Invalid ID",
			},
			Status: http.StatusBadRequest,
		}
	}

	return uapi.HttpResponse{
		Json: assets.CheckUserBoosterStatus(idSnow),
	}
}
