package get_user

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

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	userColsArr = utils.GetCols(types.User{})
	userCols    = strings.Join(userColsArr, ",")

	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	indexPackColsArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")
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

	if name == "undefined" {
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   `{"error":"false","message":"Handling known issue"}`,
		}
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "uc-"+name).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	var user types.User

	var err error

	row, err := state.Pool.Query(d.Context, "SELECT "+userCols+" FROM users WHERE user_id = $1", name)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	err = pgxscan.ScanOne(&user, row)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if utils.IsNone(user.About.String) {
		user.About.Valid = false
		user.About.String = ""
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

	var indexBots = []types.IndexBot{}

	err = pgxscan.ScanAll(&indexBots, indexBotRows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range indexBots {
		userObj, err := dovewing.GetUser(d.Context, indexBots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		indexBots[i].User = userObj
	}

	// Get user teams
	// Teams the user is a member in
	var userTeamIds []string

	userTeamRows, err := state.Pool.Query(d.Context, "SELECT team_id FROM team_members WHERE user_id = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&userTeamIds, userTeamRows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var userTeams = []types.Team{}

	for _, teamId := range userTeamIds {
		team, err := utils.ResolveTeam(d.Context, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		indexBots = append(indexBots, team.UserBots...)
		userTeams = append(userTeams, *team)
	}

	// Packs
	packsRows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs WHERE owner = $1 ORDER BY created_at DESC", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	packs := []types.IndexBotPack{}

	err = pgxscan.ScanAll(&packs, packsRows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user.UserPacks = packs
	user.UserBots = indexBots
	user.UserTeams = userTeams

	return uapi.HttpResponse{
		Json:      user,
		CacheKey:  "uc-" + name,
		CacheTime: 2 * time.Minute,
	}
}
