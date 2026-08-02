package get_server_meta

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"
	"time"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server Meta",
		Description: "Resolves a Discord invite link to a preview of the server it points to (name, icon, member counts, and whether it's already listed), without adding it to the database. Used by Add Server to show the server being added before the user submits.",
		Resp:        types.DiscordServerMeta{},
		Params: []docs.Parameter{
			{
				Name:        "invite",
				Description: "The Discord invite link or code to resolve",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "get_server_meta",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "get_server_meta"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	rawInvite := r.URL.Query().Get("invite")

	if rawInvite == "" {
		return resp.BadRequest("The invite query parameter is required")
	}

	invite, err := assets.ResolveInvite(d.Context, rawInvite)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	meta := types.DiscordServerMeta{
		ServerID:      invite.Guild.ID.String(),
		Name:          invite.Guild.Name,
		Avatar:        assets.GuildIconURL(invite.Guild),
		TotalMembers:  invite.ApproximateMemberCount,
		OnlineMembers: invite.ApproximatePresenceCount,
	}

	err = state.Pool.QueryRow(d.Context, "SELECT type FROM servers WHERE server_id = $1", meta.ServerID).Scan(&meta.ListType)

	switch {
	case err == nil:
		meta.AlreadyListed = true
	case errors.Is(err, pgx.ErrNoRows):
		// Not listed yet, expected for the common case.
	default:
		return resp.Err("Error while checking if server is already in database", err, zap.String("serverID", meta.ServerID))
	}

	presence := assets.CheckBotGuildPresence(d.Context, meta.ServerID)
	meta.BotPresent = presence.Present
	meta.BotInviteURL = presence.InviteURL

	return resp.OK(meta)
}
