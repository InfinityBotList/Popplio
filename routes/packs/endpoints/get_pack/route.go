package get_pack

import (
	"net/http"
	"strings"

	"popplio/db"
	"popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var (
	packColArr = db.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")

	indexBotColArr = db.GetCols(types.IndexBot{})
	indexBotCols   = strings.Join(indexBotColArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Pack",
		Description: "Gets a pack on the list based on the URL.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The URL of the pack.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BotPack{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	if id == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	row, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE url = $1", id)

	if err != nil {
		state.Logger.Error("Error querying packs table [db fetch]", zap.Error(err), zap.String("url", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	pack, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.BotPack])

	if err == pgx.ErrNoRows {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Error querying packs table [collect]", zap.Error(err), zap.String("url", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = assets.ResolveBotPack(d.Context, &pack)

	if err != nil {
		state.Logger.Error("Error resolving bot pack", zap.Error(err), zap.String("url", id))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error resolving bot pack: " + err.Error()},
		}
	}

	return uapi.HttpResponse{
		Json: pack,
	}
}
