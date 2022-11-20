package get_pack

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	packColArr = utils.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/packs/{id}",
		OpId:        "get_pack",
		Summary:     "Get Pack",
		Description: "Gets a pack on the list based on either URL or Name.",
		Tags:        []string{api.CurrentTag},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The ID of the pack.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BotPack{},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var id = chi.URLParam(r, "id")

	if id == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	var pack types.BotPack

	row, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE url = $1 OR name = $1", id)

	if err != nil {
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	err = pgxscan.ScanOne(&pack, row)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	err = utils.ResolveBotPack(d.Context, state.Pool, &pack)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	d.Resp <- types.HttpResponse{
		Json: pack,
	}
}
