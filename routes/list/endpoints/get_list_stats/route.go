package get_list_stats

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
)

type ListStatsBot struct {
	BotID              string   `json:"bot_id"`
	Vanity             string   `json:"vanity"`
	Short              string   `json:"short"`
	Type               string   `json:"type"`
	MainOwnerID        string   `json:"main_owner_id"`
	AdditionalOwnerIDS []string `json:"additional_owners_ids"`
	QueueName          string   `json:"queue_name"`
}

type ListStats struct {
	Bots         []ListStatsBot `json:"bots"`
	TotalStaff   int64          `json:"total_staff"`
	TotalUsers   int64          `json:"total_users"`
	TotalVotes   int64          `json:"total_votes"`
	TotalPacks   int64          `json:"total_packs"`
	TotalTickets int64          `json:"total_tickets"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/stats",
		OpId:        "get_list_stats",
		Summary:     "Get List Statistics",
		Description: "Gets the statistics of the list",
		Tags:        []string{api.CurrentTag},
		Resp: ListStats{
			Bots: []ListStatsBot{},
		},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	listStats := ListStats{}

	bots, err := state.Pool.Query(d.Context, "SELECT bot_id, vanity, short, type, owner, additional_owners, queue_name FROM bots")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	defer bots.Close()

	for bots.Next() {
		var botId string
		var vanity string
		var short string
		var typeStr string
		var owner string
		var additionalOwners []string
		var queueName string

		err := bots.Scan(&botId, &vanity, &short, &typeStr, &owner, &additionalOwners, &queueName)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		listStats.Bots = append(listStats.Bots, ListStatsBot{
			BotID:              botId,
			Vanity:             vanity,
			Short:              short,
			Type:               typeStr,
			MainOwnerID:        owner,
			AdditionalOwnerIDS: additionalOwners,
			QueueName:          queueName,
		})
	}

	var activeStaff int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users WHERE staff = true").Scan(&activeStaff)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	listStats.TotalStaff = activeStaff

	var totalUsers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	listStats.TotalUsers = totalUsers

	var totalVotes int64
	err = state.Pool.QueryRow(d.Context, "SELECT SUM(votes) FROM bots").Scan(&totalVotes)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	listStats.TotalVotes = totalVotes

	var totalPacks int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&totalPacks)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	listStats.TotalPacks = totalPacks

	var totalTickets int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM transcripts").Scan(&totalTickets)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	listStats.TotalTickets = totalTickets

	return api.HttpResponse{
		Json: listStats,
	}
}
