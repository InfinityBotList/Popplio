package get_staff_templates

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	templateTypesColsArr = db.GetCols(types.StaffTemplateType{})
	templateTypesCols    = strings.Join(templateTypesColsArr, ",")

	templateColsArr = db.GetCols(types.StaffTemplate{})
	templateCols    = strings.Join(templateColsArr, ",")
)

func Docs() *doclib.Doc {
	return &doclib.Doc{
		Summary:     "Get Staff Templates",
		Description: "Returns all of the staff templates used for reviewing bots",
		Resp:        types.StaffTemplateList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(state.Context, "SELECT "+templateCols+" FROM staff_templates ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error("Failed to fetch staff templates list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	templates, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.StaffTemplate])

	if err != nil {
		state.Logger.Error("Failed to fetch staff templates list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	typeRows, err := state.Pool.Query(state.Context, "SELECT "+templateTypesCols+" FROM staff_templates_types ORDER BY created_at DESC")

	if err != nil {
		state.Logger.Error("Failed to fetch staff templates types list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	templatesTypes, err := pgx.CollectRows(typeRows, pgx.RowToStructByName[types.StaffTemplateType])

	if err != nil {
		state.Logger.Error("Failed to fetch staff templates type list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: types.StaffTemplateList{
			Templates:     templates,
			TemplateTypes: templatesTypes,
		},
	}
}
