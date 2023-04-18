package get_team_permissions

import (
	"net/http"
	"popplio/teams"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

type PermissionResponse struct {
	Perms []teams.PermDetailMap `json:"perms"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team Permissions",
		Description: "Gets all permissions that a team can have",
		Resp:        PermissionResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: PermissionResponse{teams.TeamPermDetails},
	}
}
