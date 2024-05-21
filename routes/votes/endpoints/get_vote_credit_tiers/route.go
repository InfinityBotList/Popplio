package get_vote_credit_tiers

import (
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

var (
	voteCreditTiersColsArr = db.GetCols(types.VoteCreditTier{})
	voteCreditTiersCols    = strings.Join(voteCreditTiersColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get General Vote Credit Tiers",
		Description: "Returns a list of all currently available vote credit tiers sorted in ascending order",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.VoteCreditTier{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")

	if targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Target Type is required to use this endpoint"},
		}
	}

	rows, err := state.Pool.Exec(d.Context, "SELECT "+voteCreditTiersCols+" FROM vote_credit_tiers WHERE target_type = $1 ORDER BY position ASC", targetType)

	return uapi.HttpResponse{}
}
