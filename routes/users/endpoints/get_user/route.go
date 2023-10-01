package get_user

import (
	"errors"
	"net/http"
	"strings"

	"popplio/assets"
	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

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
	name := chi.URLParam(r, "id")

	if name == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	row, err := state.Pool.Query(d.Context, "SELECT "+userCols+" FROM users WHERE user_id = $1", name)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	user, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.User])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	userObj, err := dovewing.GetUser(d.Context, user.ID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.User = userObj

	indexBotRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE owner = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.UserBots, err = pgx.CollectRows(indexBotRows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range user.UserBots {
		userObj, err := dovewing.GetUser(d.Context, user.UserBots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		user.UserBots[i].User = userObj

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", user.UserBots[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		user.UserBots[i].Vanity = code
		user.UserBots[i].Banner = assets.BannerInfo("bots", user.UserBots[i].BotID)
	}

	// Get user teams
	// Teams the user is a member in
	userTeamRows, err := state.Pool.Query(d.Context, "SELECT team_id FROM team_members WHERE user_id = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	tids, err := pgx.CollectRows[string](userTeamRows, func(row pgx.CollectableRow) (string, error) {
		var id string
		err := row.Scan(&id)
		return id, err
	})

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, tid := range tids {
		row, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", tid)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto.Entities, err = resolvers.GetTeamEntities(d.Context, tid, []string{
			"bot",
			"server",
		})

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		eto.Banner = assets.BannerInfo("teams", eto.ID)

		user.UserTeams = append(user.UserTeams, eto)
	}

	// Packs
	packsRows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs WHERE owner = $1 ORDER BY created_at DESC", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.UserPacks, err = pgx.CollectRows(packsRows, pgx.RowToStructByName[types.IndexBotPack])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: user,
	}
}
