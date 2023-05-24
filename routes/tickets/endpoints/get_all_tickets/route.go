package get_all_tickets

import (
	"net/http"
	"strconv"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

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
		Resp: types.PagedResult[types.Ticket]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Check if user is admin
	var admin bool

	err := state.Pool.QueryRow(d.Context, "SELECT admin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&admin)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !admin {
		return uapi.HttpResponse{
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
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	// Get ticket
	var tickets []types.Ticket

	row, err := state.Pool.Query(d.Context, "SELECT "+ticketCols+" FROM tickets WHERE open = false LIMIT $1 OFFSET $2", limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&tickets, row)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	for i := range tickets {
		tickets[i].Author, err = dovewing.GetDiscordUser(d.Context, tickets[i].UserID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if tickets[i].CloseUserID.Valid && tickets[i].CloseUserID.String != "" {
			tickets[i].CloseUser, err = dovewing.GetDiscordUser(d.Context, tickets[i].CloseUserID.String)

			if err != nil {
				state.Logger.Error(err)
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}
		}

		for j := range tickets[i].Messages {
			tickets[i].Messages[j].Author, err = dovewing.GetDiscordUser(d.Context, tickets[i].Messages[j].AuthorID)

			if err != nil {
				state.Logger.Error(err)
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			// Convert snowflake ID to timestamp
			tickets[i].Messages[j].Timestamp, err = discordgo.SnowflakeTimestamp(tickets[i].Messages[j].ID)

			if err != nil {
				state.Logger.Error(err)
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}
		}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM tickets").Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[types.Ticket]{
		Count:   count,
		PerPage: perPage,
		Results: tickets,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
