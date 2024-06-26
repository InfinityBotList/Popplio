package get_list_stats

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
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
		state.Logger.Error("Failed to fetch bot count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalApprovedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'approved'").Scan(&totalApprovedBots)

	if err != nil {
		state.Logger.Error("Failed to fetch approved bot count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalCertifiedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'certified'").Scan(&totalCertifiedBots)

	if err != nil {
		state.Logger.Error("Failed to fetch certified bot count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalStaff int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM staff_members").Scan(&totalStaff)

	if err != nil {
		state.Logger.Error("Failed to fetch user count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalUsers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

	if err != nil {
		state.Logger.Error("Failed to fetch total user count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalVotes int64
	err = state.Pool.QueryRow(d.Context, "SELECT SUM(approximate_votes) FROM bots").Scan(&totalVotes)

	if err != nil {
		state.Logger.Error("Failed to fetch total vote count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalPacks int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&totalPacks)

	if err != nil {
		state.Logger.Error("Failed to fetch total pack count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var totalTickets int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM tickets").Scan(&totalTickets)

	if err != nil {
		state.Logger.Error("Failed to fetch total ticket count", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
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
