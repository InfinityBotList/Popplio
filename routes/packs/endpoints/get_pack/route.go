package get_pack

import (
	"context"
	"net/http"
	"strings"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	packColArr = utils.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	if id == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	// First check count so we can avoid expensive DB calls
	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", id).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var pack types.BotPack

	row, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE url = $1", id)

	if err != nil {
		return api.DefaultResponse(http.StatusNotFound)
	}

	err = pgxscan.ScanOne(&pack, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = ResolveBotPack(d.Context, &pack)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: pack,
	}
}

func ResolveBotPack(ctx context.Context, pack *types.BotPack) error {
	ownerUser, err := utils.GetDiscordUser(ctx, pack.Owner)

	if err != nil {
		return err
	}

	pack.Votes, err = utils.ResolvePackVotes(ctx, pack.URL)

	if err != nil {
		return err
	}

	pack.ResolvedOwner = ownerUser

	for _, botId := range pack.Bots {
		var short string
		var bot_type pgtype.Text
		var vanity pgtype.Text
		var banner pgtype.Text
		var nsfw bool
		var premium bool
		var shards int
		var votes int
		var inviteClicks int
		var servers int
		var tags []string
		err := state.Pool.QueryRow(ctx, "SELECT short, type, vanity, banner, nsfw, premium, shards, votes, invite_clicks, servers, tags FROM bots WHERE bot_id = $1", botId).Scan(&short, &bot_type, &vanity, &banner, &nsfw, &premium, &shards, &votes, &inviteClicks, &servers, &tags)

		if err == pgx.ErrNoRows {
			continue
		}

		if err != nil {
			return err
		}

		botUser, err := utils.GetDiscordUser(ctx, botId)

		if err != nil {
			return err
		}

		pack.ResolvedBots = append(pack.ResolvedBots, types.ResolvedPackBot{
			Short:        short,
			User:         botUser,
			Type:         bot_type,
			Vanity:       vanity,
			Banner:       banner,
			NSFW:         nsfw,
			Premium:      premium,
			Shards:       shards,
			Votes:        votes,
			InviteClicks: inviteClicks,
			Servers:      servers,
			Tags:         tags,
		})
	}

	return nil
}
