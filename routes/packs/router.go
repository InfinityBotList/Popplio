package packs

import (
	"popplio/api"
	"popplio/routes/packs/endpoints/add_pack"
	"popplio/routes/packs/endpoints/get_all_packs"
	"popplio/routes/packs/endpoints/get_pack"
	"popplio/routes/packs/endpoints/patch_pack"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Bot Packs"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to IBL packs"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/packs/{id}",
		OpId:    "get_pack",
		Method:  api.GET,
		Docs:    get_pack.Docs,
		Handler: get_pack.Route,
	}.Route(r)

	api.Route{
		Pattern: "/packs/all",
		OpId:    "get_all_packs",
		Method:  api.GET,
		Docs:    get_all_packs.Docs,
		Handler: get_all_packs.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/packs",
		OpId:    "add_pack",
		Method:  api.PUT,
		Docs:    add_pack.Docs,
		Handler: add_pack.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   types.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/packs/{id}",
		OpId:    "patch_pack",
		Method:  api.PATCH,
		Docs:    patch_pack.Docs,
		Handler: patch_pack.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   types.TargetTypeUser,
			},
		},
	}.Route(r)
}
