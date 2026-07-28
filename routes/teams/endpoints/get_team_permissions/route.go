package get_team_permissions

import (
	"net/http"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team Permissions",
		Description: "Gets all permissions that a team can have",
		Resp:        types.PermissionResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: types.PermissionResponse{Perms: teams.PermDetails},
	}
}
