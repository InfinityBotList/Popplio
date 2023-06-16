package packs

import (
	"popplio/api"
	"popplio/routes/packs/endpoints/add_pack"
	"popplio/routes/packs/endpoints/delete_pack"
	"popplio/routes/packs/endpoints/get_all_packs"
	"popplio/routes/packs/endpoints/get_pack"
	"popplio/routes/packs/endpoints/get_pack_seo"
	"popplio/routes/packs/endpoints/patch_pack"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const (
	tagName = "Bot Packs"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to IBL packs"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/packs/{id}",
		OpId:    "get_pack",
		Method:  uapi.GET,
		Docs:    get_pack.Docs,
		Handler: get_pack.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/packs/{id}/seo",
		OpId:    "get_pack_seo",
		Method:  uapi.GET,
		Docs:    get_pack_seo.Docs,
		Handler: get_pack_seo.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/packs/@all",
		OpId:    "get_all_packs",
		Method:  uapi.GET,
		Docs:    get_all_packs.Docs,
		Handler: get_all_packs.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/packs",
		OpId:    "add_pack",
		Method:  uapi.PUT,
		Docs:    add_pack.Docs,
		Handler: add_pack.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/packs/{id}",
		OpId:    "patch_pack",
		Method:  uapi.PATCH,
		Docs:    patch_pack.Docs,
		Handler: patch_pack.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/packs/{id}",
		OpId:    "delete_pack",
		Method:  uapi.DELETE,
		Docs:    delete_pack.Docs,
		Handler: delete_pack.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
