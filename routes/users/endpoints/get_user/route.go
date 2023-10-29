package get_user

import (
	"errors"
	"net/http"
	"strings"

	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"

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

	indexPackColsArr = db.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")

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
		state.Logger.Error("Error while getting user [db fetch]", zap.Error(err), zap.String("userID", userId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	userObj, err := dovewing.GetUser(d.Context, user.ID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Error while getting user [collect]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.User = userObj

	indexBotRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE owner = $1", user.ID)

	if err != nil {
		state.Logger.Error("Failed to get user bots [db fetch]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.UserBots, err = pgx.CollectRows(indexBotRows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error("Failed to get user bots [collect]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range user.UserBots {
		userObj, err := dovewing.GetUser(d.Context, user.UserBots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Error while getting bot user [dovewing]", zap.Error(err), zap.String("botID", user.UserBots[i].BotID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		user.UserBots[i].User = userObj

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", user.UserBots[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error("Error while getting vanity code", zap.Error(err), zap.String("botID", user.UserBots[i].BotID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		user.UserBots[i].Vanity = code
		user.UserBots[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeBots, user.UserBots[i].BotID)
	}

	// Get user teams
	// Teams the user is a member in
	userTeamRows, err := state.Pool.Query(d.Context, "SELECT team_id FROM team_members WHERE user_id = $1", user.ID)

	if err != nil {
		state.Logger.Error("Error while getting user teams [db fetch]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	tids, err := pgx.CollectRows[string](userTeamRows, func(row pgx.CollectableRow) (string, error) {
		var id string
		err := row.Scan(&id)
		return id, err
	})

	if err != nil {
		state.Logger.Error("Error while getting user teams [collect]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, tid := range tids {
		row, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", tid)

		if err != nil {
			state.Logger.Error("Error while getting team [db fetch]", zap.Error(err), zap.String("teamID", tid), zap.String("userID", user.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

		if err != nil {
			state.Logger.Error("Error while getting team [collect]", zap.Error(err), zap.String("teamID", tid), zap.String("userID", user.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto.Entities, err = resolvers.GetTeamEntities(d.Context, tid, []string{
			"bot",
			"server",
		})

		if err != nil {
			state.Logger.Error("Error while getting team entities", zap.Error(err), zap.String("teamID", tid), zap.String("userID", user.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeTeams, eto.ID)
		eto.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, eto.ID)

		user.UserTeams = append(user.UserTeams, eto)
	}

	// Packs
	packsRows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs WHERE owner = $1 ORDER BY created_at DESC", user.ID)

	if err != nil {
		state.Logger.Error("Error while getting user packs [db fetch]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.UserPacks, err = pgx.CollectRows(packsRows, pgx.RowToStructByName[types.IndexBotPack])

	if err != nil {
		state.Logger.Error("Error while getting user packs [collect]", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: user,
	}
}
