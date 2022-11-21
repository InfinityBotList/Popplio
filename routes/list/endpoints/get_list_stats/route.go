package get_list_stats

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/stats",
		OpId:        "get_list_stats",
		Summary:     "Get List Statistics",
		Description: "Gets the statistics of the list",
		Tags:        []string{api.CurrentTag},
		Resp: types.ListStats{
			Bots: []types.ListStatsBot{},
		},
	})
}

func Route(d api.RouteData, r *http.Request) {
	listStats := types.ListStats{}

	bots, err := state.Pool.Query(d.Context, "SELECT bot_id, vanity, short, type, owner, additional_owners, certified, claimed FROM bots")

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	defer bots.Close()

	for bots.Next() {
		var botId string
		var vanity string
		var short string
		var typeStr string
		var owner string
		var additionalOwners []string
		var certified bool
		var claimed bool

		err := bots.Scan(&botId, &vanity, &short, &typeStr, &owner, &additionalOwners, &certified, &claimed)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
			return
		}

		listStats.Bots = append(listStats.Bots, types.ListStatsBot{
			BotID:              botId,
			Vanity:             vanity,
			Short:              short,
			Type:               typeStr,
			MainOwnerID:        owner,
			AdditionalOwnerIDS: additionalOwners,
			Certified:          certified,
			Claimed:            claimed,
		})
	}

	var activeStaff int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users WHERE staff = true").Scan(&activeStaff)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listStats.TotalStaff = activeStaff

	var totalUsers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listStats.TotalUsers = totalUsers

	var totalVotes int64
	err = state.Pool.QueryRow(d.Context, "SELECT SUM(votes) FROM bots").Scan(&totalVotes)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listStats.TotalVotes = totalVotes

	var totalPacks int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&totalPacks)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listStats.TotalPacks = totalPacks

	var totalTickets int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM transcripts").Scan(&totalTickets)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	listStats.TotalTickets = totalTickets

	d.Resp <- api.HttpResponse{
		Json: listStats,
	}
}
