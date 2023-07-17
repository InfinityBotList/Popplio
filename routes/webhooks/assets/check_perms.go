package assets

import (
	"context"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/infinitybotlist/eureka/uapi"
)

type Operation int

const (
	OpWebhookLogs  Operation = iota
	OpTestWebhooks Operation = iota
)

type perm struct {
	botPerm  types.TeamPermission `validate:"required"`
	teamPerm types.TeamPermission `validate:"required"`
}

func (o Operation) Perms() perm {
	switch o {
	case OpWebhookLogs:
		return perm{
			botPerm:  teams.TeamPermissionGetBotWebhookLogs,
			teamPerm: teams.TeamPermissionGetTeamWebhookLogs,
		}
	case OpTestWebhooks:
		return perm{
			botPerm:  teams.TeamPermissionTestBotWebhooks,
			teamPerm: teams.TeamPermissionTestTeamWebhooks,
		}
	default:
		return perm{}
	}
}

func CheckWebhookPermissions(
	ctx context.Context,
	targetId,
	targetType,
	userId string,
	o Operation,
) (resp uapi.HttpResponse, ok bool) {
	p := o.Perms()

	err := state.Validator.Struct(p)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError), false
	}

	switch targetType {
	case "bot":
		var count int

		err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError), false
		}

		if count == 0 {
			return uapi.DefaultResponse(http.StatusNotFound), false
		}

		perms, err := utils.GetUserBotPerms(ctx, userId, targetId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError), false
		}

		if !perms.Has(p.botPerm) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to get bot webhook logs"},
			}, false
		}
	case "team":
		var count int

		err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM teams WHERE id = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError), false
		}

		if count == 0 {
			return uapi.DefaultResponse(http.StatusNotFound), false
		}

		// Ensure manager is a member of the team
		var managerCount int

		err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", targetId, userId).Scan(&managerCount)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError), false
		}

		if managerCount == 0 {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You are not a member of this team"},
			}, false
		}

		var managerPerms []types.TeamPermission
		err = state.Pool.QueryRow(ctx, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", targetId, userId).Scan(&managerPerms)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError), false
		}

		mp := teams.NewPermissionManager(managerPerms)

		if !mp.Has(p.teamPerm) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to get team webhook logs"},
			}, false
		}

	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "This entity type is not supported yet"},
		}, false
	}

	return uapi.HttpResponse{}, true
}
