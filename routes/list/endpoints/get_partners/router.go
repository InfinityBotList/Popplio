package get_partners

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

var (
	// Partners
	partnersColsArr = db.GetCols(types.Partner{})
	partnersCols    = strings.Join(partnersColsArr, ",")

	// Partner types
	partnerTypesColsArr = db.GetCols(types.PartnerTypes{})
	partnerTypesCols    = strings.Join(partnerTypesColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Partners",
		Description: "Gets the official partners of the list",
		Resp:        types.PartnerList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(state.Context, "SELECT "+partnersCols+" FROM partners ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	partners, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Partner])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range partners {
		err := state.Validator.Struct(partners[i])

		if err != nil {
			state.Logger.Error(err)
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Could not validate " + partners[i].ID + " with error:" + err.Error()},
			}
		}

		partners[i].User, err = dovewing.GetUser(state.Context, partners[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	rows, err = state.Pool.Query(state.Context, "SELECT "+partnerTypesCols+" FROM partner_types ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	partnerTypes, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.PartnerTypes])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.PartnerList{
			Partners:     partners,
			PartnerTypes: partnerTypes,
		},
	}
}
