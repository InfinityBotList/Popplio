package get_user

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	userColsArr = utils.GetCols(types.User{})
	userCols    = strings.Join(userColsArr, ",")

	userBotColsArr = utils.GetCols(types.UserBot{})
	userBotCols    = strings.Join(userBotColsArr, ",")

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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	if name == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	if name == "undefined" {
		return api.HttpResponse{
			Status: http.StatusOK,
			Data:   `{"error":"false","message":"Handling known issue"}`,
		}
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "uc-"+name).Val()
	if cache != "" {
		return api.HttpResponse{
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
		return api.DefaultResponse(http.StatusNotFound)
	}

	err = pgxscan.ScanOne(&user, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if utils.IsNone(user.About.String) {
		user.About.Valid = false
		user.About.String = ""
	}

	userObj, err := dovewing.GetDiscordUser(d.Context, user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	user.User = userObj

	userBotsRows, err := state.Pool.Query(d.Context, "SELECT "+userBotCols+" FROM bots WHERE owner = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var userBots = []types.UserBot{}

	err = pgxscan.ScanAll(&userBots, userBotsRows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range userBots {
		userObj, err := dovewing.GetDiscordUser(d.Context, userBots[i].BotID)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		userBots[i].User = userObj
	}

	/*


	 */

	// Get user teams
	// Teams the user is a member in
	var userTeamIds []string

	userTeamRows, err := state.Pool.Query(d.Context, "SELECT team_id FROM team_members WHERE user_id = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&userTeamIds, userTeamRows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var userTeams = []types.Team{}

	for _, teamId := range userTeamIds {
		team, err := utils.ResolveTeam(d.Context, teamId)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		userBots = append(userBots, team.UserBots...)
		userTeams = append(userTeams, *team)
	}

	/*


	 */

	// Packs
	packsRows, err := state.Pool.Query(d.Context, "SELECT "+indexPackCols+" FROM packs WHERE owner = $1 ORDER BY created_at DESC", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	packs := []types.IndexBotPack{}

	err = pgxscan.ScanAll(&packs, packsRows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range packs {
		packs[i].Votes, err = utils.ResolvePackVotes(d.Context, packs[i].URL)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	user.UserPacks = packs
	user.UserBots = userBots
	user.UserTeams = userTeams

	return api.HttpResponse{
		Json:      user,
		CacheKey:  "uc-" + name,
		CacheTime: 2 * time.Minute,
	}
}
