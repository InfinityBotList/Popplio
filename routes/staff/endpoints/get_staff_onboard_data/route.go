package get_staff_onboard_data

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Staff Onboard Data",
		Description: "Gets the staff onboarding data based on an Onboarding ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Onboarding ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.StaffOnboardData{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := chi.URLParam(r, "id")

	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM onboard_data WHERE onboard_code = $1", id).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var dataMap map[string]any
	var userId string

	err = state.Pool.QueryRow(d.Context, "SELECT user_id, data FROM onboard_data WHERE onboard_code = $1", id).Scan(&userId, &dataMap)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: types.StaffOnboardData{
			UserID: userId,
			Data:   dataMap,
		},
	}
}
