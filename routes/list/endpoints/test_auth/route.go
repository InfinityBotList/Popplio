package test_auth

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/types"
)

type TestAuth struct {
	AuthType types.TargetType `json:"auth_type"`
	TargetID string           `json:"target_id"`
	Token    string           `json:"token"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/list/test-auth",
		OpId:        "test_auth",
		Summary:     "Test Auth",
		Description: "Test your authentication",
		Req:         TestAuth{},
		Resp:        api.AuthData{},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var payload TestAuth

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		d.Resp <- hresp
		return
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
		d.Resp <- hr
		return
	}

	// Check if the auth type is correct
	if authData.TargetType != payload.AuthType {
		d.Resp <- api.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid auth type"},
		}
		return
	}

	// Check if the auth target id is correct
	if payload.TargetID != "" && authData.ID != payload.TargetID {
		d.Resp <- api.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid auth target id"},
		}
		return
	}

	d.Resp <- api.HttpResponse{
		Json: authData,
	}
}
