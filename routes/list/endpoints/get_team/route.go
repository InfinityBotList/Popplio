package get_team

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	userPermColsArr = utils.GetCols(types.UserPerm{})
	userPermCols    = strings.Join(userPermColsArr, ",")
)

type StaffTeam struct {
	Members []types.UserPerm `json:"members"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Staff Team",
		Description: "Gets an up to date listing of the staff team of the list",
		Resp:        StaffTeam{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+userPermCols+" FROM users WHERE staff = true")

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var users = []types.UserPerm{}

	err = pgxscan.ScanAll(&users, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i, user := range users {
		user, err := utils.GetDiscordUser(d.Context, user.ID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		users[i].User = user
	}

	return api.HttpResponse{
		Status: http.StatusOK,
		Json: StaffTeam{
			Members: users,
		},
	}
}
