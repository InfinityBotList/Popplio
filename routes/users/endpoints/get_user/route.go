package get_user

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slices"
)

var (
	userColsArr = utils.GetCols(types.User{})
	userCols    = strings.Join(userColsArr, ",")

	userBotColsArr = utils.GetCols(types.UserBot{})
	userBotCols    = strings.Join(userBotColsArr, ",")

	indexPackColsArr = utils.GetCols(types.IndexBotPack{})
	indexPackCols    = strings.Join(indexPackColsArr, ",")

	userTeamColsArr = utils.GetCols(types.UserTeam{})
	userTeamCols    = strings.Join(userTeamColsArr, ",")
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

	userObj, err := utils.GetDiscordUser(d.Context, user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	user.User = userObj

	userBotsRows, err := state.Pool.Query(d.Context, "SELECT "+userBotCols+" FROM bots WHERE owner = $1 OR additional_owners && $2", user.ID, []string{user.ID})

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

	parsedUserBots := []types.UserBot{}
	for _, bot := range userBots {
		userObj, err := utils.GetDiscordUser(d.Context, bot.BotID)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		bot.User = userObj
		parsedUserBots = append(parsedUserBots, bot)
	}

	user.UserBots = parsedUserBots

	// Get user teams

	// Teams the user is a owner in
	var userOwnerTeams []string

	userOwnerTeamRows, err := state.Pool.Query(d.Context, "SELECT id FROM teams WHERE owner = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&userOwnerTeams, userOwnerTeamRows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Teams the user is a member in
	var userTeams []string

	userTeamRows, err := state.Pool.Query(d.Context, "SELECT team_id FROM team_members WHERE user_id = $1", user.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&userTeams, userTeamRows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Merge the two slices
	for _, teamId := range userOwnerTeams {
		if !slices.Contains(userTeams, teamId) {
			userTeams = append(userTeams, teamId)
		}
	}

	var teams = []types.UserTeam{}

	for _, teamId := range userTeams {
		var team = types.UserTeam{}

		teamBotsRows, err := state.Pool.Query(d.Context, "SELECT "+userTeamCols+" FROM teams WHERE id = $1", teamId)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		err = pgxscan.ScanOne(&team, teamBotsRows)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		teams = append(teams, team)
	}

	user.UserTeams = teams

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

	return api.HttpResponse{
		Json:      user,
		CacheKey:  "uc-" + name,
		CacheTime: 2 * time.Minute,
	}
}
