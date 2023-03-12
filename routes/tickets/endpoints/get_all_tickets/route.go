package get_all_tickets

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
)

const perPage = 5

var (
	ticketColsArr = utils.GetCols(types.Ticket{})
	ticketCols    = strings.Join(ticketColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Tickets",
		Description: "Gets a support ticket. Requires admin permissions",
		Params: []docs.Parameter{
			{
				Name:        "page",
				In:          "query",
				Description: "The page of tickets to get. Defaults to 1",
				Required:    false,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Ticket{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Check if user is admin
	var admin bool

	err := state.Pool.QueryRow(d.Context, "SELECT admin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&admin)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !admin {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You do not have permission to view this ticket",
				Error:   true,
			},
		}
	}

	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	// Get ticket
	var ticket []types.Ticket

	row, err := state.Pool.Query(d.Context, "SELECT "+ticketCols+" FROM tickets WHERE open = false LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&ticket, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	for i := range ticket {
		ticket[i].Author, err = utils.GetDiscordUser(d.Context, ticket[i].UserID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		if ticket[i].CloseUserID.Valid && ticket[i].CloseUserID.String != "" {
			ticket[i].CloseUser, err = utils.GetDiscordUser(d.Context, ticket[i].CloseUserID.String)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}
		}

		for j := range ticket[i].Messages {
			ticket[i].Messages[j].Author, err = utils.GetDiscordUser(d.Context, ticket[i].Messages[j].AuthorID)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}

			// Convert snowflake ID to timestamp
			ticket[i].Messages[j].Timestamp, err = discordgo.SnowflakeTimestamp(ticket[i].Messages[j].ID)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}
		}
	}

	var previous strings.Builder

	// More optimized string concat
	previous.WriteString(state.Config.Sites.API)
	previous.WriteString("/bots/all?page=")
	previous.WriteString(strconv.FormatUint(pageNum-1, 10))

	if pageNum-1 < 1 || pageNum == 0 {
		previous.Reset()
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM tickets").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var next strings.Builder

	next.WriteString(state.Config.Sites.API)
	next.WriteString("/bots/all?page=")
	next.WriteString(strconv.FormatUint(pageNum+1, 10))

	if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
		next.Reset()
	}

	data := types.AllTickets{
		Count:    count,
		Results:  ticket,
		PerPage:  perPage,
		Previous: previous.String(),
		Next:     next.String(),
	}

	return api.HttpResponse{
		Json: data,
	}
}
