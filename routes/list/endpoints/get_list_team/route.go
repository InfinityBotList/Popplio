package get_list_team

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	userPermColsArr = db.GetCols(types.UserPerm{})
	userPermCols    = strings.Join(userPermColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Team",
		Description: "Gets an up to date listing of the staff team of the list. This is currently broken and does not handle permissions yet (TODO)",
		Resp:        types.StaffTeam{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Not yet fully implemented
	sms, err := state.Pool.Query(d.Context, "SELECT user_id FROM staff_members")

	if err != nil {
		state.Logger.Error("Failed to fetch staff team [sms]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer sms.Close()

	var staffIds []string

	for sms.Next() {
		var id string
		err := sms.Scan(&id)

		if err != nil {
			state.Logger.Error("Failed to fetch staff team [sms]", zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		staffIds = append(staffIds, id)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+userPermCols+" FROM users WHERE user_id = ANY($1)", staffIds)

	if err != nil {
		state.Logger.Error("Failed to fetch staff team [rows]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.UserPerm])

	if err != nil {
		state.Logger.Error("Failed to fetch staff team [collect]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i, user := range users {
		user, err := dovewing.GetUser(d.Context, user.ID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch staff team member [dovewing]", zap.Error(err), zap.String("id", user.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		users[i].User = user
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.StaffTeam{
			Members: users,
		},
	}
}
