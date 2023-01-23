package test_auth

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/types"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Auth",
		Description: "Test your authentication",
		Req:         types.TestAuth{},
		Resp:        api.AuthData{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload types.TestAuth

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if payload.TargetID == "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Target ID is required"},
		}
	}

	// Create []AuthType

	authType := []api.AuthType{
		{
			Type: payload.AuthType,
		},
	}

	r.Header.Set("Authorization", payload.Token)

	// Check auth
	authData, hr, ok := api.Route{
		Auth: authType,
	}.Authorize(r)

	if !ok {
		return hr
	}

	// Check if the auth type is correct
	if authData.TargetType != payload.AuthType {
		return api.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid auth type"},
		}
	}

	return api.HttpResponse{
		Json: authData,
	}
}
