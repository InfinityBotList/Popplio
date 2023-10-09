package get_changelog

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

var (
	changelogColsArr = db.GetCols(&types.Changelog{})
	changelogCols    = strings.Join(changelogColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Changelog",
		Description: "Gets the changelog of the list",
		Resp:        types.Changelog{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+changelogCols+" FROM changelogs ORDER BY version DESC")

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	changelogs, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Changelog])

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   changelogs,
	}
}
