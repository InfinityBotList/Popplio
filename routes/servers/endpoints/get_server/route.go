package get_server

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

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
- invite -> server invite view
- vote -> server vote page`,
				Required: false,
				In:       "query",
				Schema:   docs.IdSchema,
			},
			{
				Name:        "short",
				Description: "Avoid sending large fields. Currently this is only the long description of the bot",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func handleAnalytics(r *http.Request, id, target string) {
	switch target {
	case "page":
		// Get IP from request and hash it
		hashedIp := fmt.Sprintf("%x", sha256.Sum256([]byte(r.RemoteAddr)))

		// Create transaction
		tx, err := state.Pool.Begin(state.Context)

		if err != nil {
			state.Logger.Error(err)
			return
		}

		defer tx.Rollback(state.Context)

		_, err = tx.Exec(state.Context, "UPDATE servers SET clicks = clicks + 1 WHERE server_id = $1", id)

		if err != nil {
			state.Logger.Error(err)
			return
		}

		// Check if the IP has already clicked the server by checking the unique_clicks row
		var hasClicked bool

		err = tx.QueryRow(state.Context, "SELECT $1 = ANY(unique_clicks) FROM servers WHERE server_id = $2", hashedIp, id).Scan(&hasClicked)

		if err != nil {
			state.Logger.Error("Error checking", err)
			return
		}

		if !hasClicked {
			// If not, add it to the array
			state.Logger.Info("Adding click for " + id)
			_, err = tx.Exec(state.Context, "UPDATE servers SET unique_clicks = array_append(unique_clicks, $1) WHERE server_id = $2", hashedIp, id)

			if err != nil {
				state.Logger.Error("Error adding:", err)
				return
			}
		}

		// Commit transaction
		err = tx.Commit(state.Context)

		if err != nil {
			state.Logger.Error(err)
			return
		}
	case "invite":
		// Update clicks
		_, err := state.Pool.Exec(state.Context, "UPDATE servers SET invite_clicks = invite_clicks + 1 WHERE server_id = $1", id)

		if err != nil {
			state.Logger.Error(err)
		}
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	target := r.URL.Query().Get("target")

	row, err := state.Pool.Query(d.Context, "SELECT "+serverCols+" FROM servers WHERE server_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	server, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Server])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !strings.HasPrefix(server.Banner.String, "https://") {
		server.Banner.Valid = false
		server.Banner.String = ""
	}

	row, err = state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", server.TeamOwnerID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	eto, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	eto.Entities = &types.TeamEntities{
		Targets: []string{}, // We don't provide any entities right now, may change
	}

	server.TeamOwner = &eto

	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(unique_clicks) AS unique_clicks FROM servers WHERE server_id = $1", server.ServerID).Scan(&uniqueClicks)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	server.UniqueClicks = uniqueClicks

	var code string

	err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", server.VanityRef).Scan(&code)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	server.Vanity = code

	go handleAnalytics(r, id, target)

	if r.URL.Query().Get("short") == "true" {
		server.Long = ""
	}

	return uapi.HttpResponse{
		Json: server,
	}
}
