package check_booster_status

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"golang.org/x/exp/slices"

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

	// Check member is a booster
	m, err := state.Discord.State.Member(state.Config.Servers.Main, id)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json: types.BoosterStatus{
				Remark:    "Member not found on server:" + err.Error(),
				IsBooster: false,
			},
		}
	}

	// Check if member has booster role
	roles := state.Config.Roles.PremiumRoles.Parse()
	for _, role := range m.Roles {
		if slices.Contains(roles, role) {
			// Member has booster role
			return uapi.HttpResponse{
				Json: types.BoosterStatus{
					IsBooster: true,
				},
			}
		}
	}

	return uapi.HttpResponse{
		Json: types.BoosterStatus{
			IsBooster: false,
		},
	}
}
