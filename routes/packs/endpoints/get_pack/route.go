package get_pack

import (
	"errors"
	"net/http"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var (
	packColArr = utils.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")

	indexBotColArr = utils.GetCols(types.IndexBot{})
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
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	pack, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.BotPack])

	if err == pgx.ErrNoRows {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	ownerUser, err := dovewing.GetUser(d.Context, pack.Owner, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	pack.ResolvedOwner = ownerUser

	for _, botId := range pack.Bots {
		row, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE bot_id = $1", botId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bot, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.IndexBot])

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", bot.VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		botUser, err := dovewing.GetUser(d.Context, botId, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bot.User = botUser

		pack.ResolvedBots = append(pack.ResolvedBots, bot)
	}

	return uapi.HttpResponse{
		Json: pack,
	}
}
