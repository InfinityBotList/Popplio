// Package get_server implements GET /servers/{id} — "Get Server".
//
// The target page of the request if any.
package get_server

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/validators"
	"popplio/votes"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	serverColsArr = db.GetCols(types.Server{})
	serverCols    = strings.Join(serverColsArr, ",")

	teamColsArr = db.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server",
		Description: "Gets a server by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The servers ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name: "target",
				Description: `The target page of the request if any. 
				
If target is 'page', then unique clicks will be counted based on a SHA-256 hashed IP

If target is 'invite', then the invite will be counted as a click

Officially recognized targets:

- page -> server page view
- settings -> server settings page view
- invite -> server invite view`,
				Required: false,
				In:       "query",
				Schema:   docs.IdSchema,
			},
			{
				Name:        "include",
				Description: "What extra fields to include, comma-seperated.\n`long` => server long description",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "team_includes",
				Description: "What entities of the servers team to include",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func handleAnalytics(r *http.Request, id, target string) error {
	switch target {
	case "page":
		// Get IP from request and hash it
		hashedIp := fmt.Sprintf("%x", sha256.Sum256([]byte(r.RemoteAddr)))

		// Create transaction
		tx, err := state.Pool.Begin(state.Context)

		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}

		defer tx.Rollback(state.Context)

		_, err = tx.Exec(state.Context, "UPDATE servers SET clicks = clicks + 1 WHERE server_id = $1", id)

		if err != nil {
			return fmt.Errorf("error updating clicks count: %w", err)
		}

		// Check if the IP has already clicked the server by checking the unique_clicks row
		var hasClicked bool

		err = tx.QueryRow(state.Context, "SELECT $1 = ANY(unique_clicks) FROM servers WHERE server_id = $2", hashedIp, id).Scan(&hasClicked)

		if err != nil {
			return fmt.Errorf("error checking for any unique clicks from this user: %w", err)
		}

		if !hasClicked {
			// If not, add it to the array
			state.Logger.Debug("Adding new unique click for user during handleAnalytics", zap.Error(err), zap.String("id", id), zap.String("target", target), zap.String("targetType", "bot"))
			_, err = tx.Exec(state.Context, "UPDATE servers SET unique_clicks = array_append(unique_clicks, $1) WHERE server_id = $2", hashedIp, id)

			if err != nil {
				return fmt.Errorf("error adding new unique click for user: %w", err)
			}
		}

		// Commit transaction
		err = tx.Commit(state.Context)

		if err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
		}
	case "invite":
		// Update clicks
		_, err := state.Pool.Exec(state.Context, "UPDATE servers SET invite_clicks = invite_clicks + 1 WHERE server_id = $1", id)

		if err != nil {
			return fmt.Errorf("error updating invite clicks: %w", err)
		}
	}

	return nil
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	target := r.URL.Query().Get("target")

	row, err := state.Pool.Query(d.Context, "SELECT "+serverCols+" FROM servers WHERE server_id = $1", id)

	if err != nil {
		return resp.Err("Error while getting server [db fetch]", err, zap.String("id", id), zap.String("target", target))
	}

	server, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Server])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting server [db collect]", err, zap.String("id", id), zap.String("target", target))
	}

	row, err = state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", server.TeamOwnerID)

	if err != nil {
		return resp.Err("Error while getting team [db fetch]", err, zap.String("id", id), zap.String("target", target))
	}

	eto, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

	if err != nil {
		return resp.Err("Error while getting team [db collect]", err, zap.String("id", id), zap.String("target", target))
	}

	if r.URL.Query().Get("team_includes") != "" {
		includesSplit := strings.Split(r.URL.Query().Get("team_includes"), ",")

		if len(includesSplit) > 16 {
			return resp.BadRequest("Too many `team_includes`. Maximum is 16")
		}

		eto.Entities, err = resolvers.GetTeamEntities(d.Context, eto.ID, includesSplit)

		if err != nil {
			return resp.ErrDetail("Error while getting team entities", err, zap.String("id", id), zap.String("target", target), zap.String("teamOwner", validators.EncodeUUID(server.TeamOwnerID.Bytes)))
		}
	} else {
		eto.Entities = &types.TeamEntities{
			Targets: []string{}, // We don't provide any entities right now, may change
		}
	}

	server.TeamOwner = &eto

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM servers WHERE server_id = $1", server.ServerID).Scan(&uniqueClicks)

	if err != nil {
		return resp.Err("Error while getting unique clicks", err, zap.String("id", id), zap.String("target", target))
	}

	server.UniqueClicks = uniqueClicks

	var code string

	err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", server.VanityRef).Scan(&code)

	if err != nil {
		return resp.Err("Error while getting bot vanity code [db collect]", err, zap.String("id", id), zap.String("target", target), zap.String("serverID", server.ServerID))
	}

	server.Vanity = code

	// The owner may have synced emoji/sticker data sitting in the DB from
	// when show_emojis was previously on — don't leak it once they've opted
	// back out, rather than waiting for the next sync pass to clear it.
	if !server.ShowEmojis {
		server.Emojis = nil
		server.Stickers = nil
	}
	// Nil Go slices serialize to JSON null, which crashes frontend consumers
	// that call .length/.map on it without a null check.
	if server.Emojis == nil {
		server.Emojis = []types.Emoji{}
	}
	if server.Stickers == nil {
		server.Stickers = []types.Sticker{}
	}

	server.Votes, err = votes.EntityGetVoteCount(d.Context, state.Pool, server.ServerID, "server")

	if err != nil {
		return resp.ErrBody("Error while getting server vote count [db fetch]", "Error while getting server vote count [db fetch].", err)
	}

	// Handle extra includes
	if r.URL.Query().Get("include") != "" {
		includesSplit := strings.Split(r.URL.Query().Get("include"), ",")

		for _, include := range includesSplit {
			switch include {
			case "long":
				// Fetch long description
				var long string
				err := state.Pool.QueryRow(d.Context, "SELECT long FROM servers WHERE server_id = $1", server.ServerID).Scan(&long)

				if err != nil {
					return resp.Err("Error while getting bot server description [db fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("serverID", server.ServerID))
				}

				server.Long = long
			}
		}
	}

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while handling analytics", zap.Any("panic", rec), zap.String("id", id), zap.String("target", target))
			}
		}()

		if err := handleAnalytics(r, id, target); err != nil {
			state.Logger.Error("Error while handling analytics", zap.Error(err), zap.String("id", id), zap.String("target", target))
		}
	}()

	return uapi.HttpResponse{
		Json: server,
	}
}
