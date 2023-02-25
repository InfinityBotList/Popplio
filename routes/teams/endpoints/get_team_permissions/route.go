package get_team_permissions

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/teams"
)

type PermissionResponse struct {
	Perms []teams.PermDetailMap `json:"perms"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team Permissions",
		Description: "Gets all permissions that the team can have",
		Resp:        PermissionResponse{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Json: PermissionResponse{teams.TeamPermDetails},
	}
}
