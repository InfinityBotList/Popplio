package assets

import (
	"popplio/api"
	"popplio/routes/assets/upload_asset"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Assets"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to assets (images etc.) on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/users/{uid}/assets/{target_id}",
		OpId:    "upload_asset",
		Method:  uapi.POST,
		Docs:    upload_asset.Docs,
		Handler: upload_asset.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
