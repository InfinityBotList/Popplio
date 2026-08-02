// Package get_ticket implements GET /tickets/{id} — "Get Ticket".
//
// Gets a support ticket. Requires you to be the author of the ticket or have
// the 'staff' permission
package get_ticket

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/snowflake/v2"
	perms "github.com/infinitybotlist/kittycat/go"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	ticketColsArr = db.GetCols(types.Ticket{})
	ticketCols    = strings.Join(ticketColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Ticket",
		Description: "Gets a support ticket. Requires you to be the author of the ticket or have the 'staff' permission",
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
		return resp.Err("Error getting ticket", err, zap.String("ticket_id", ticketId))
	}

	if userId != d.Auth.ID {
		// Check if they are staff with popplio.tickets permission
		sp, err := validators.GetUserStaffPerms(d.Context, d.Auth.ID)

		if err != nil {
			return resp.ErrBody("Failed to get user staff perms", "Failed to get user staff perms.", err)
		}

		if !perms.HasPerm(sp.Resolve(), perms.Permission{Namespace: "popplio", Perm: "tickets"}) {
			return resp.Forbidden("You do not have permission to view this ticket [popplio.tickets is required]")
		}
	}

	// Get ticket
	row, err := state.Pool.Query(d.Context, "SELECT "+ticketCols+" FROM tickets WHERE id = $1", ticketId)

	if err != nil {
		return resp.Err("Error getting ticket [db fetch]", err, zap.String("ticket_id", ticketId))
	}

	ticket, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Ticket])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error getting ticket [collect]", err, zap.String("ticket_id", ticketId))
	}

	// Parse the ticket
	ticket.Author, err = dovewing.GetUser(d.Context, ticket.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error getting ticket author [dovewing]", err, zap.String("ticket_id", ticketId))
	}

	if ticket.CloseUserID.Valid && ticket.CloseUserID.String != "" {
		ticket.CloseUser, err = dovewing.GetUser(d.Context, ticket.CloseUserID.String, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error getting ticket closer [dovewing]", err, zap.String("ticket_id", ticketId))
		}
	}

	for i := range ticket.Messages {
		ticket.Messages[i].Author, err = dovewing.GetUser(d.Context, ticket.Messages[i].AuthorID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error getting ticket message author [dovewing]", err, zap.String("ticket_id", ticketId))
		}

		// Convert snowflake ID to timestamp
		id, err := snowflake.Parse(ticket.Messages[i].ID)

		if err != nil {
			return resp.Err("Error parsing snowflake", err, zap.String("ticket_id", ticketId))
		}

		ticket.Messages[i].Timestamp = id.Time()
	}

	return uapi.HttpResponse{
		Json: ticket,
	}
}
