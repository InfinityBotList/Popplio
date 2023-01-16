package get_ticket

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

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
		Description: "Gets a support ticket. **Note that this endpoint is only documented to be useful for staff and the like. It is not useful for normal users**",
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	ticketId := chi.URLParam(r, "id")

	if ticketId == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "tik-"+ticketId).Val()
	if cache != "" {
		return api.HttpResponse{
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
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&ticket, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Parse the ticket
	ticket.Author, err = utils.GetDiscordUser(ticket.UserID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if ticket.CloseUserID.Valid && ticket.CloseUserID.String != "" {
		ticket.CloseUser, err = utils.GetDiscordUser(ticket.CloseUserID.String)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	for i := range ticket.Messages {
		ticket.Messages[i].Author, err = utils.GetDiscordUser(ticket.Messages[i].AuthorID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		// Convert snowflake ID to timestamp
		ticket.Messages[i].Timestamp, err = discordgo.SnowflakeTimestamp(ticket.Messages[i].ID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.HttpResponse{
		Json:      ticket,
		CacheKey:  "tik-" + ticketId,
		CacheTime: time.Minute * 3,
	}
}
