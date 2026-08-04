// Package get_staff_permissions implements GET /staff/meta/permissions — "Get
// Staff Permissions".
//
// Lists every permission that can be attached to a staff role, which is what a
// permission picker needs to render before anyone has chosen anything.
//
// Like its team counterpart at /teams/meta/permissions this is unauthenticated:
// it returns the vocabulary, not anyone's permissions, and holding a name grants
// nothing. Who may actually attach one of these to a role is decided when the
// role is edited, by the panel and the staff bot.
package get_staff_permissions

import (
	"net/http"

	"popplio/perms"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Staff Permissions",
		Description: "Lists every permission that can be attached to a staff role, grouped by category.",
		Resp:        types.PermissionResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: types.PermissionResponse{
			Perms: types.PermissionDataFrom(perms.Staff.Definitions()),
		},
	}
}
