package get_team_permissions

import (
	"net/http"
	"popplio/api"
	"popplio/teams"

	docs "github.com/infinitybotlist/doclib"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Json: PermissionResponse{teams.TeamPermDetails},
	}
}
