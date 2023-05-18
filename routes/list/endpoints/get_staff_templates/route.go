package get_staff_templates

import (
	"net/http"
	"popplio/stafftemplates"
	"popplio/types"

	"github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *doclib.Doc {
	return &doclib.Doc{
		Summary:     "Get Staff Templates",
		Description: "Returns all of the staff templates used for reviewing bots",
		Resp:        types.StaffTemplateList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   stafftemplates.StaffTemplates,
	}
}
