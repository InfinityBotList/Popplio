package get_pack

import (
	"context"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

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
		d.Resp <- api.DefaultResponse(http.StatusBadRequest)
		return
	}

	var pack types.BotPack

	row, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE url = $1 OR name = $1", id)

	if err != nil {
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	err = pgxscan.ScanOne(&pack, row)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	err = ResolveBotPack(d.Context, &pack)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	d.Resp <- api.HttpResponse{
		Json: pack,
	}
}

func ResolveBotPack(ctx context.Context, pack *types.BotPack) error {
	ownerUser, err := utils.GetDiscordUser(pack.Owner)

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

		botUser, err := utils.GetDiscordUser(botId)

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
