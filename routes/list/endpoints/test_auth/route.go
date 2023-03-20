package test_auth

import (
	"context"
	"net/http"

	"popplio/api"
	"popplio/types"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

type TestAuth struct {
	AuthType api.TargetType `json:"auth_type"`
	TargetID string         `json:"target_id"`
	Token    string         `json:"token"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Auth",
		Description: "Test your authentication",
		Req:         TestAuth{},
		Resp:        api.AuthData{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload TestAuth

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if payload.TargetID == "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Target ID is required", Error: true},
		}
	}

	// Create []AuthType
	rctx := context.Background()
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("test", payload.TargetID)
	rctx = context.WithValue(rctx, chi.RouteCtxKey, ctx)
	authType := []api.AuthType{
		{
			URLVar: "test",
			Type:   payload.AuthType,
		},
	}

	reqCtxd := r.WithContext(rctx)

	r.Header.Set("Authorization", payload.Token)

	// Check auth
	authData, hr, ok := api.Route{
		Auth: authType,
	}.Authorize(reqCtxd)

	if !ok {
		return hr
	}

	return api.HttpResponse{
		Json: authData,
	}
}
