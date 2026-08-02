// Package get_list_stats implements GET /list/stats — "Get List Statistics".
//
// Gets basic statistics of the list
package get_list_stats

import (
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Statistics",
		Description: "Gets basic statistics of the list",
		Resp:        types.ListStats{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var totalBots int64
	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&totalBots)

	if err != nil {
		return resp.Err("Failed to fetch bot count", err)
	}

	var totalApprovedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'approved'").Scan(&totalApprovedBots)

	if err != nil {
		return resp.Err("Failed to fetch approved bot count", err)
	}

	var totalCertifiedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'certified'").Scan(&totalCertifiedBots)

	if err != nil {
		return resp.Err("Failed to fetch certified bot count", err)
	}

	var totalStaff int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM staff_members").Scan(&totalStaff)

	if err != nil {
		return resp.Err("Failed to fetch user count", err)
	}

	var totalUsers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

	if err != nil {
		return resp.Err("Failed to fetch total user count", err)
	}

	var totalVotes int64
	err = state.Pool.QueryRow(d.Context, "SELECT SUM(approximate_votes) FROM bots").Scan(&totalVotes)

	if err != nil {
		return resp.Err("Failed to fetch total vote count", err)
	}

	var totalPacks int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&totalPacks)

	if err != nil {
		return resp.Err("Failed to fetch total pack count", err)
	}

	var totalTickets int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM tickets").Scan(&totalTickets)

	if err != nil {
		return resp.Err("Failed to fetch total ticket count", err)
	}

	return uapi.HttpResponse{
		Json: types.ListStats{
			TotalBots:          totalBots,
			TotalApprovedBots:  totalApprovedBots,
			TotalCertifiedBots: totalCertifiedBots,
			TotalStaff:         totalStaff,
			TotalUsers:         totalUsers,
			TotalVotes:         totalVotes,
			TotalPacks:         totalPacks,
			TotalTickets:       totalTickets,
		},
	}
}
