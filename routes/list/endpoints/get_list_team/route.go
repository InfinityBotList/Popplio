package get_list_team

import (
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

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
		Summary:     "Get List Team",
		Description: "Gets an up to date listing of the staff team of the list",
		Resp:        StaffTeam{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+userPermCols+" FROM users WHERE staff = true")

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var users = []types.UserPerm{}

	err = pgxscan.ScanAll(&users, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i, user := range users {
		user, err := dovewing.GetDiscordUser(d.Context, user.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		users[i].User = user
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: StaffTeam{
			Members: users,
		},
	}
}
