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
	"go.uber.org/zap"
)

var (
	changelogEntryColsArr = db.GetCols(types.ChangelogEntry{})
	changelogEntryCols    = strings.Join(changelogEntryColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Changelog",
		Description: "Gets the changelog of the list",
		Resp:        types.Changelog{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+changelogEntryCols+" FROM changelogs ORDER BY version DESC")

	if err != nil {
		state.Logger.Error("Failed to fetch changelog entries [rows]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	changelogs, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ChangelogEntry])

	if err != nil {
		state.Logger.Error("Failed to fetch changelog entries [collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.Changelog{
			Entries: changelogs,
		},
	}
}
