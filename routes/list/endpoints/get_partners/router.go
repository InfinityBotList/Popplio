package get_partners

import (
	"net/http"
	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
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
		state.Logger.Error("Failed to fetch partner list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	partners, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Partner])

	if err != nil {
		state.Logger.Error("Failed to fetch partner list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range partners {
		err := state.Validator.Struct(partners[i])

		if err != nil {
			state.Logger.Error("Failed to validate partner", zap.Error(err), zap.String("partner_id", partners[i].ID))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Could not validate " + partners[i].ID + " with error:" + err.Error()},
			}
		}

		partners[i].User, err = dovewing.GetUser(state.Context, partners[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch partner user", zap.Error(err), zap.String("partner_id", partners[i].ID), zap.String("user_id", partners[i].UserID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		partners[i].Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypePartners, partners[i].ID)
	}

	rows, err = state.Pool.Query(state.Context, "SELECT "+partnerTypesCols+" FROM partner_types ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error("Failed to fetch partner types [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	partnerTypes, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.PartnerTypes])

	if err != nil {
		state.Logger.Error("Failed to fetch partner types [db fetch]", zap.Error(err))
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
