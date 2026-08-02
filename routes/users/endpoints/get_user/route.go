// Package get_user implements GET /users/{id} — "Get User".
//
// Gets a user by id
package get_user

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	"popplio/routes/packs/assets"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/votes"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	userColsArr = db.GetCols(types.User{})
	userCols    = strings.Join(userColsArr, ",")

	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	packColsArr = db.GetCols(types.BotPack{})
	packCols    = strings.Join(packColsArr, ",")

	teamColsArr = db.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User",
		Description: "Gets a user by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.User{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	userId := chi.URLParam(r, "id")

	if userId == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	row, err := state.Pool.Query(d.Context, "SELECT "+userCols+" FROM users WHERE user_id = $1", userId)

	if err != nil {
		state.Logger.Error("Error while getting user", zap.Error(err), zap.String("userID", userId))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	user, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.User])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting user [db fetch]", err, zap.String("userID", userId))
	}

	userObj, err := dovewing.GetUser(d.Context, user.ID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error while getting user [collect]", err, zap.String("userID", user.ID))
	}

	user.User = userObj

	indexBotRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE owner = $1", user.ID)

	if err != nil {
		return resp.Err("Failed to get user bots [db fetch]", err, zap.String("userID", user.ID))
	}

	user.UserBots, err = pgx.CollectRows(indexBotRows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		return resp.Err("Failed to get user bots [collect]", err, zap.String("userID", user.ID))
	}

	// Resolve the userbots concurrently, since each bot's resolution is independent
	if err := botAssets.ResolveIndexBots(d.Context, user.UserBots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	// Get user teams
	// Teams the user is a member in
	userTeamRows, err := state.Pool.Query(d.Context, "SELECT team_id FROM team_members WHERE user_id = $1", user.ID)

	if err != nil {
		return resp.Err("Error while getting user teams [db fetch]", err, zap.String("userID", user.ID))
	}

	tids, err := pgx.CollectRows[string](userTeamRows, func(row pgx.CollectableRow) (string, error) {
		var id string
		err := row.Scan(&id)
		return id, err
	})

	if err != nil {
		return resp.Err("Error while getting user teams [collect]", err, zap.String("userID", user.ID))
	}

	// Ensure this always marshals as `[]` rather than `null` when the user
	// isn't on any teams — a nil Go slice serializes to JSON null, which
	// crashes frontend consumers that call .length/.map on it without a
	// null check.
	user.UserTeams = []types.Team{}

	for _, tid := range tids {
		row, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", tid)

		if err != nil {
			return resp.Err("Error while getting team [db fetch]", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		eto, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

		if err != nil {
			return resp.Err("Error while getting team [collect]", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		eto.Entities, err = resolvers.GetTeamEntities(d.Context, tid, []string{
			"team_member",
			"bot",
			"server",
		})

		if err != nil {
			return resp.Err("Error while getting team entities", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		// Votes is db:"-" (resolved in application code, not scanned from the
		// row above) — without this, every team embedded here would silently
		// report 0 votes instead of the real count that GET /teams/{id} shows.
		eto.Votes, err = votes.EntityGetVoteCount(d.Context, state.Pool, tid, "team")

		if err != nil {
			return resp.Err("Error while getting team vote count", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		user.UserTeams = append(user.UserTeams, eto)
	}

	// Packs
	packsRows, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE owner = $1 ORDER BY created_at DESC", user.ID)

	if err != nil {
		return resp.Err("Error while getting user packs [db fetch]", err, zap.String("userID", user.ID))
	}

	user.UserPacks, err = pgx.CollectRows(packsRows, pgx.RowToStructByName[types.BotPack])

	if err != nil {
		return resp.Err("Error while getting user packs [collect]", err, zap.String("userID", user.ID))
	}

	for i := range user.UserPacks {
		err = assets.ResolveBotPack(d.Context, &user.UserPacks[i])

		if err != nil {
			return resp.ErrBody("Error while resolving user pack", "Error resolving user pack.", err, zap.String("userID", user.ID), zap.String("url", user.UserPacks[i].URL))
		}
	}

	// Fetch staff status
	var positions int

	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(positions) FROM staff_members WHERE user_id = $1", user.ID).Scan(&positions)

	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return resp.ErrBody("Error while getting staff status", "Error getting staff status.", err, zap.String("userID", user.ID))
	}

	user.Staff = positions > 0

	return uapi.HttpResponse{
		Json: user,
	}
}
