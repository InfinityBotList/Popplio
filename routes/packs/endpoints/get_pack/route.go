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
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type BotPack struct {
	Owner         string             `db:"owner" json:"owner_id"`
	ResolvedOwner *types.DiscordUser `db:"-" json:"owner"`
	Name          string             `db:"name" json:"name"`
	Short         string             `db:"short" json:"short"`
	Votes         []types.PackVote   `db:"-" json:"votes"`
	Tags          []string           `db:"tags" json:"tags"`
	URL           string             `db:"url" json:"url"`
	CreatedAt     time.Time          `db:"created_at" json:"created_at"`
	Bots          []string           `db:"bots" json:"bot_ids"`
	ResolvedBots  []ResolvedPackBot  `db:"-" json:"bots"`
}

type ResolvedPackBot struct {
	User         *types.DiscordUser `json:"user"`
	Short        string             `json:"short"`
	Type         pgtype.Text        `json:"type"`
	Vanity       pgtype.Text        `json:"vanity"`
	Banner       pgtype.Text        `json:"banner"`
	NSFW         bool               `json:"nsfw"`
	Premium      bool               `json:"premium"`
	Shards       int                `json:"shards"`
	Votes        int                `json:"votes"`
	InviteClicks int                `json:"invites"`
	Servers      int                `json:"servers"`
	Tags         []string           `json:"tags"`
}

var (
	packColArr = utils.GetCols(BotPack{})
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
		Resp: BotPack{},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	if id == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	var pack BotPack

	row, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE url = $1 OR name = $1", id)

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

func ResolveBotPack(ctx context.Context, pack *BotPack) error {
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

		pack.ResolvedBots = append(pack.ResolvedBots, ResolvedPackBot{
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
