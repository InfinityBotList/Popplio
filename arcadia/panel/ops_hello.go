package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

// The panel's opening handshake and the dashboard numbers behind it.
//
// hello is what the frontend calls first: it carries the protocol version, the
// instance's own configuration and the caller's staff record, so a panel that
// cannot complete it cannot render anything.

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
			Warnings:    []string{},
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
