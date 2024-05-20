package get_entity_permissions

import (
	"net/http"

	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Entity Permissions",
		Description: "Returns the resolved permissions a user has on an entity",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserEntityPerms{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "id")
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	perms, err := teams.GetEntityPerms(d.Context, uid, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error getting entity perms", zap.Error(err), zap.String("uid", uid), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	return uapi.HttpResponse{
		Json: types.UserEntityPerms{
			Perms: func() []string {
				var fperms []string
				for _, perm := range perms {
					fperms = append(fperms, perm.String())
				}
				return fperms
			}(),
		},
	}
}
