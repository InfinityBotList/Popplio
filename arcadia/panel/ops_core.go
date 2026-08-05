package panel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/rpc"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// helloVersion is the protocol version the panel must send.
const helloVersion = 5

// checkAuth wraps the auth check so failures surface the way Error::new does:
// status 500 with the bare message the frontend matches on.
func checkAuth(ctx context.Context, token string) (types.AuthData, error) {
	data, err := impls.CheckAuth(ctx, token)

	if err != nil {
		return types.AuthData{}, newError(err)
	}

	return data, nil
}

func checkAuthInsecure(ctx context.Context, token string) (types.AuthData, error) {
	data, err := impls.CheckAuthInsecure(ctx, token)

	if err != nil {
		return types.AuthData{}, newError(err)
	}

	return data, nil
}

// resolvedPerms is the light permission path used by most operations.
func resolvedPerms(ctx context.Context, userID string) (perms.Set, error) {
	sp, err := impls.GetUserPerms(ctx, userID)

	if err != nil {
		return perms.Set{}, newError(err)
	}

	return sp, nil
}

func (s *Server) hello(ctx context.Context, q *types.QHello) (response, error) {
	// Note the order: auth is validated BEFORE the version check.
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if q.Version != helloVersion {
		return writeText(http.StatusBadRequest, "Invalid version"), nil
	}

	staffMember, err := impls.GetStaffMember(ctx, authData.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, types.Hello{
		InstanceConfig: types.InstanceConfig{
			Description: instanceDescription(),
			Warnings: []string{},
		},
		AuthData:    authData,
		StaffMember: staffMember,
		CoreConstants: types.CoreConstants{
			FrontendURL:    state.Config.Sites.Frontend.Parse(),
			InfernoplexURL: state.Config.Sites.Infernoplex.Parse(),
			PopplioURL:     state.Config.Sites.API.Parse(),
			Servers:        serverIDs(),
		},
		TargetTypes: types.TargetTypeVariants,
	}), nil
}

func (s *Server) baseAnalytics(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	botCounts, err := groupedCounts(ctx, "SELECT type, COUNT(*) FROM bots GROUP BY type")

	if err != nil {
		return response{}, newError(err)
	}

	serverCounts, err := groupedCounts(ctx, "SELECT type, COUNT(*) FROM servers GROUP BY type")

	if err != nil {
		return response{}, newError(err)
	}

	ticketCounts, err := ticketCounts(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	var totalUsers int64

	if err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers); err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, types.BaseAnalytics{
		BotCounts:    botCounts,
		ServerCounts: serverCounts,
		TicketCounts: ticketCounts,
		TotalUsers:   totalUsers,
		// Hardcoded 0 upstream.
		ChangelogsCount: 0,
	}), nil
}

func groupedCounts(ctx context.Context, query string) (map[string]int64, error) {
	rows, err := state.Pool.Query(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	counts := make(map[string]int64)

	for rows.Next() {
		var (
			key   string
			count int64
		)

		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}

		counts[key] = count
	}

	return counts, rows.Err()
}

// ticketCounts groups tickets by the boolean `open` column, which the response
// exposes as the keys "open" and "closed".
func ticketCounts(ctx context.Context) (map[string]int64, error) {
	rows, err := state.Pool.Query(ctx, "SELECT open, COUNT(*) FROM tickets GROUP BY open")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	counts := make(map[string]int64)

	for rows.Next() {
		var (
			open  bool
			count int64
		)

		if err := rows.Scan(&open, &count); err != nil {
			return nil, err
		}

		if open {
			counts["open"] = count
		} else {
			counts["closed"] = count
		}
	}

	return counts, rows.Err()
}

func (s *Server) getUser(ctx context.Context, q *types.QGetUser) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	user, err := impls.GetPlatformUser(ctx, q.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, user), nil
}

type botQueueRow struct {
	BotID            string     `db:"bot_id"`
	ClientID         string     `db:"client_id"`
	LastClaimed      *time.Time `db:"last_claimed"`
	ClaimedBy        *string    `db:"claimed_by"`
	Type             string     `db:"type"`
	ApprovalNote     string     `db:"approval_note"`
	Short            string     `db:"short"`
	Invite           string     `db:"invite"`
	ApproximateVotes int32      `db:"approximate_votes"`
	Shards           int32      `db:"shards"`
	Library          string     `db:"library"`
	InviteClicks     int32      `db:"invite_clicks"`
	Clicks           int32      `db:"clicks"`
	Servers          int32      `db:"servers"`
}

func (s *Server) botQueue(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	// Public to all staff: no permission check.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	rows, err := state.Pool.Query(ctx,
		`SELECT bot_id, client_id, last_claimed, claimed_by, type, approval_note, short,
                invite, approximate_votes, shards, library, invite_clicks, clicks, servers
                FROM bots WHERE type = 'pending' OR type = 'claimed' ORDER BY created_at`)

	if err != nil {
		return response{}, newError(err)
	}

	queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[botQueueRow])

	if err != nil {
		return response{}, newError(err)
	}

	return s.partialBots(ctx, queue)
}

// partialBots renders a bot list.
//
// Upstream resolved each bot's managers with two queries inside the loop, so an
// N-bot response cost 2N round trips. The managers are batched here into three
// queries total; the emitted JSON and the per-bot error messages are unchanged.
func (s *Server) partialBots(ctx context.Context, queue []botQueueRow) (response, error) {
	botIDs := make([]string, 0, len(queue))

	for _, bot := range queue {
		botIDs = append(botIDs, bot.BotID)
	}

	managers, err := impls.GetBotManagers(ctx, botIDs)

	if err != nil {
		return response{}, newError(err)
	}

	bots := make([]types.PartialEntity, 0, len(queue))

	for _, bot := range queue {
		// Dovewing is fronted by a Redis hot cache, so this stays per-bot.
		user, err := impls.GetPlatformUser(ctx, bot.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		bots = append(bots, types.PartialEntity{Bot: &types.PartialBot{
			BotID:        bot.BotID,
			ClientID:     bot.ClientID,
			User:         user,
			ClaimedBy:    bot.ClaimedBy,
			LastClaimed:  types.TimestampPtr(bot.LastClaimed),
			ApprovalNote: bot.ApprovalNote,
			Short:        bot.Short,
			Type:         bot.Type,
			Votes:        bot.ApproximateVotes,
			Shards:       bot.Shards,
			Library:      bot.Library,
			InviteClicks: bot.InviteClicks,
			Clicks:       bot.Clicks,
			Servers:      bot.Servers,
			Mentionable:  managers[bot.BotID].Mentionables(),
			Invite:       bot.Invite,
		}})
	}

	return writeJSON(http.StatusOK, bots), nil
}

func (s *Server) executeRpc(ctx context.Context, q *types.QExecuteRpc) (response, error) {
	// The endpoint is open to all staff; the RPC layer enforces per-method perms.
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	resp, rpcErr := rpc.Execute(ctx, q.Method, rpc.Handle{
		UserID:     authData.UserID,
		TargetType: q.TargetType,
	})

	if rpcErr != nil {
		// RPC failures are 400, not 500.
		return writeText(http.StatusBadRequest, rpcErr.Error()), nil
	}

	if content, ok := resp.Text(); ok {
		return writeText(http.StatusOK, content), nil
	}

	return writeNoContent(), nil
}

func (s *Server) getRpcMethods(ctx context.Context, q *types.QGetRpcMethods) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return response{}, err
	}

	actions := make([]types.RPCWebAction, 0, len(types.RPCMethodVariants))

	for _, name := range types.RPCMethodVariants {
		variant, err := types.EmptyRPCMethod(name)

		if err != nil {
			return response{}, newError(err)
		}

		required, known := types.RPCPermission(name)

		if q.Filtered && (!known || !userPerms.Has(required)) {
			continue
		}

		actions = append(actions, types.RPCWebAction{
			ID:                   name,
			Label:                variant.Label(),
			Description:          variant.Description(),
			SupportedTargetTypes: variant.SupportedTargetTypes(),
			Fields:               variant.Fields(),
		})
	}

	return writeJSON(http.StatusOK, actions), nil
}

type rpcLogRow struct {
	ID        pgtype.UUID `db:"id"`
	UserID    string      `db:"user_id"`
	Method    string      `db:"method"`
	Data      []byte      `db:"data"`
	State     string      `db:"state"`
	CreatedAt time.Time   `db:"created_at"`
}

func (s *Server) getRpcLogEntries(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return response{}, err
	}

	if !userPerms.Has(perms.StaffViewAuditLogs) {
		return writeText(http.StatusForbidden, "You do not have permission to view rpc logs [view_audit_logs]"), nil
	}

	rows, err := state.Pool.Query(ctx, "SELECT id, user_id, method, data, state, created_at FROM rpc_logs ORDER BY created_at DESC")

	if err != nil {
		return response{}, newError(err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[rpcLogRow])

	if err != nil {
		return response{}, newError(err)
	}

	log := make([]types.RPCLogEntry, 0, len(entries))

	for _, entry := range entries {
		log = append(log, types.RPCLogEntry{
			ID:        impls.UUIDString(entry.ID),
			UserID:    entry.UserID,
			Method:    entry.Method,
			Data:      entry.Data,
			State:     entry.State,
			CreatedAt: types.NewTimestamp(entry.CreatedAt),
		})
	}

	return writeJSON(http.StatusOK, log), nil
}

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

// popplioStaff is a signed reverse proxy into Popplio's staff API.
func (s *Server) popplioStaff(ctx context.Context, q *types.QPopplioStaff) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !validHTTPMethod(q.Method) {
		return writeText(http.StatusBadRequest, "Invalid method"), nil
	}

	if !strings.HasPrefix(q.Path, "/") {
		return writeText(http.StatusBadRequest, "Path must start with /"), nil
	}

	// HARDENING (§14b): upstream only checks the leading '/', so a path of
	// "//evil.example" or one containing ".." could retarget the request at a
	// different host or escape the API base. We resolve the path against the base
	// URL and reject anything that leaves it. Legitimate paths (a plain absolute
	// path, optionally with a query string) are unaffected.
	target, err := safeJoinPopplio(q.Path)

	if err != nil {
		return writeText(http.StatusBadRequest, "Path must start with /"), nil
	}

	var popplioToken string

	err = state.Pool.QueryRow(ctx, "SELECT popplio_token FROM staffpanel__authchain WHERE token = $1", q.LoginToken).Scan(&popplioToken)

	if err != nil {
		return response{}, newError(err)
	}

	req, err := http.NewRequestWithContext(ctx, q.Method, target, strings.NewReader(q.Body))

	if err != nil {
		return response{}, newError(err)
	}

	req.Header.Set("User-Agent", "arcadia")
	// QUIRK (reproduced): the request PATH is sent as X-Forwarded-For. This looks
	// like a copy/paste bug but downstream may rely on it. See CONFORMANCE.md.
	req.Header.Set("X-Forwarded-For", q.Path)
	req.Header.Set("X-Staff-Auth-Token", popplioToken)
	req.Header.Set("X-User-ID", authData.UserID)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return response{}, newError(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return response{}, newError(err)
	}

	// Relay the upstream status and body verbatim.
	return writeText(resp.StatusCode, string(body)), nil
}

func validHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect,
		http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
