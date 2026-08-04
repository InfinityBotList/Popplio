// Package clear_user implements DELETE /platform/user/{id} — "Clear Platform
// User Cache".
//
// This endpoint will clear the cache for a user id on a given platform. This
// is useful if the user's data has changes
package clear_user

import (
	"net/http"
	"popplio/api/resp"

	"popplio/state"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Clear Platform User Cache",
		Description: "This endpoint will clear the cache for a user id on a given platform. This is useful if the user's data has changes",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
			{
				Name:        "platform",
				In:          "query",
				Description: "The platform to get the user from.",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: dovewing.ClearUserInfo{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")
	var platform = r.URL.Query().Get("platform")

	var dovewingPlatform dovewing.Platform

	switch platform {
	case "discord":
		dovewingPlatform = state.DovewingPlatformDiscord
	default:
		return resp.Status(http.StatusUnsupportedMediaType, "Unsupported platform. Only `discord` is supported at this time as a platform.")
	}

	res, err := dovewing.ClearUser(d.Context, id, dovewingPlatform, dovewing.ClearUserReq{})

	if err != nil {
		state.Logger.Error("Error clearing user [dovewing]", zap.Error(err), zap.String("id", id), zap.String("platform", platform))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	return uapi.HttpResponse{
		Json: res,
	}
}
