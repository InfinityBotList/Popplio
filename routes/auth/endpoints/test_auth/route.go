// Package test_auth implements POST /auth/test — "Test Auth".
//
// Test your authentication
package test_auth

import (
	"context"
	"net/http"
	"popplio/api/resp"

	"popplio/api"
	"popplio/perms"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Auth",
		Description: "Test your authentication",
		Req:         types.TestAuth{},
		Resp:        uapi.AuthData{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.TestAuth

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if payload.TargetID == "" {
		return resp.BadRequest("Target ID is required")
	}

	// Create []AuthType
	rctx := context.Background()
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("test", payload.TargetID)
	rctx = context.WithValue(rctx, chi.RouteCtxKey, ctx)
	authType := []uapi.AuthType{
		{
			URLVar:       "test",
			Type:         payload.AuthType,
			AllowedScope: "ban_exempt",
		},
	}

	r.Header.Set("Authorization", payload.Token)
	reqCtxd := r.WithContext(rctx)

	// Check auth
	//
	// api.Authorize reads PERMISSION_CHECK_KEY out of the route's ExtData
	// unconditionally (even when the auth type has no permission model, like
	// "user") — an empty ExtData here previously made every call, valid
	// token or not, fail with a 500 ("permissionCheck not found in
	// route.ExtData") before ever reaching the actual auth check. This is a
	// pure "is this token valid" test, so NeededPermission always returns
	// nil, which short-circuits past the permission-check block entirely.
	authData, hr, ok := api.Authorize(uapi.Route{
		Auth: authType,
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(uapi.Route, *http.Request, uapi.AuthData) (*perms.Perm, error) {
					return nil, nil
				},
			},
		},
	}, reqCtxd)

	if !ok {
		return hr
	}

	return uapi.HttpResponse{
		Json: authData,
	}
}
