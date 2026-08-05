package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// Entity search, one query shape per target type.

type searchServerRow struct {
	ServerID         string     `db:"server_id"`
	Name             string     `db:"name"`
	Avatar           string     `db:"avatar"`
	TotalMembers     int32      `db:"total_members"`
	OnlineMembers    int32      `db:"online_members"`
	Short            string     `db:"short"`
	Type             string     `db:"type"`
	ApproximateVotes int32      `db:"approximate_votes"`
	InviteClicks     int32      `db:"invite_clicks"`
	Clicks           int32      `db:"clicks"`
	NSFW             bool       `db:"nsfw"`
	Tags             []string   `db:"tags"`
	Premium          bool       `db:"premium"`
	ClaimedBy        *string    `db:"claimed_by"`
	LastClaimed      *time.Time `db:"last_claimed"`
}

func (s *Server) searchEntitys(ctx context.Context, q *types.QSearchEntitys) (response, error) {
	// No permission check: open to all staff.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	pattern := "%" + q.Query + "%"

	switch q.TargetType {
	case types.TargetTypeBot:
		rows, err := state.Pool.Query(ctx,
			`SELECT bot_id, client_id, type, approximate_votes, shards, library, invite_clicks, clicks,
                        servers, last_claimed, claimed_by, approval_note, short, invite FROM bots
                        INNER JOIN internal_user_cache__discord discord_users ON bots.bot_id = discord_users.id
                        WHERE bot_id = $1 OR client_id = $1 OR discord_users.username ILIKE $2 ORDER BY bots.created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[botQueueRow])

		if err != nil {
			return response{}, newError(err)
		}

		return s.partialBots(ctx, queue)
	case types.TargetTypeServer:
		rows, err := state.Pool.Query(ctx,
			`SELECT server_id, name, avatar, total_members, online_members, short, type, approximate_votes, invite_clicks,
                        clicks, nsfw, tags, premium, claimed_by, last_claimed FROM servers
                        WHERE server_id = $1 OR name ILIKE $2 ORDER BY created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[searchServerRow])

		if err != nil {
			return response{}, newError(err)
		}

		serverIDs := make([]string, 0, len(queue))

		for _, server := range queue {
			serverIDs = append(serverIDs, server.ServerID)
		}

		managers, err := impls.GetServerManagers(ctx, serverIDs)

		if err != nil {
			return response{}, newError(err)
		}

		servers := make([]types.PartialEntity, 0, len(queue))

		for _, server := range queue {
			servers = append(servers, types.PartialEntity{Server: &types.PartialServer{
				ServerID: server.ServerID,
				Name:     server.Name,
				// Populated from servers.avatar, synced by Infernoplex's
				// serversync task while it's a member of the guild. Empty
				// until the first sync (or if the bot has never joined).
				Avatar:        server.Avatar,
				TotalMembers:  server.TotalMembers,
				OnlineMembers: server.OnlineMembers,
				Short:         server.Short,
				Type:          server.Type,
				Votes:         server.ApproximateVotes,
				InviteClicks:  server.InviteClicks,
				Clicks:        server.Clicks,
				NSFW:          server.NSFW,
				Tags:          types.NonNilStrings(server.Tags),
				Premium:       server.Premium,
				ClaimedBy:     server.ClaimedBy,
				LastClaimed:   types.TimestampPtr(server.LastClaimed),
				Mentionable:   managers[server.ServerID].Mentionables(),
			}})
		}

		return writeJSON(http.StatusOK, servers), nil
	default:
		return writeText(http.StatusNotImplemented, "Searching this target type is not implemented"), nil
	}
}
