package service_directory

import (
	"net/http"

	"popplio/srvdirectory"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Public Service Directory",
		Description: "Returns a list of available public services",
		Resp:        types.ServiceDiscovery{},
		Params: []docs.Parameter{
			{
				Name:        "directory",
				Description: "The directory to lookup",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	directory := chi.URLParam(r, "directory")

	if directory == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Directory is required"},
		}
	}

	dir, ok := srvdirectory.Directory[directory]

	if !ok {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Directory not found"},
		}
	}

	return uapi.HttpResponse{
		Json: dir,
	}
}
