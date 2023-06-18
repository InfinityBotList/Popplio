package get_ticket

import (
	"net/http"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	ticketColsArr = utils.GetCols(types.Ticket{})
	ticketCols    = strings.Join(ticketColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Ticket",
		Description: "Gets a support ticket. Requires you to be the author of the ticket or staff",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The ticket's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Ticket{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	ticketId := chi.URLParam(r, "id")

	if ticketId == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Check ownership
	var userId string

	err := state.Pool.QueryRow(d.Context, "SELECT user_id FROM tickets WHERE id = $1", ticketId).Scan(&userId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if userId != d.Auth.ID {
		// Check if user is staff
		var staff bool

		err = state.Pool.QueryRow(d.Context, "SELECT staff FROM users WHERE user_id = $1", d.Auth.ID).Scan(&staff)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if !staff {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json: types.ApiError{
					Message: "You do not have permission to view this ticket",
					Error:   true,
				},
			}
		}
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "tik-"+ticketId).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	// Get ticket
	var ticket types.Ticket

	row, err := state.Pool.Query(d.Context, "SELECT "+ticketCols+" FROM tickets WHERE id = $1", ticketId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&ticket, row)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Parse the ticket
	ticket.Author, err = dovewing.GetUser(d.Context, ticket.UserID, state.Discord)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if ticket.CloseUserID.Valid && ticket.CloseUserID.String != "" {
		ticket.CloseUser, err = dovewing.GetUser(d.Context, ticket.CloseUserID.String, state.Discord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	for i := range ticket.Messages {
		ticket.Messages[i].Author, err = dovewing.GetUser(d.Context, ticket.Messages[i].AuthorID, state.Discord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		// Convert snowflake ID to timestamp
		ticket.Messages[i].Timestamp, err = discordgo.SnowflakeTimestamp(ticket.Messages[i].ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return uapi.HttpResponse{
		Json:      ticket,
		CacheKey:  "tik-" + ticketId,
		CacheTime: time.Minute * 3,
	}
}
