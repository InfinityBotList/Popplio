package packs

import (
	"popplio/api"
	"popplio/routes/packs/endpoints/get_all_packs"
	"popplio/routes/packs/endpoints/get_pack"

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
	r.Route("/packs", func(r chi.Router) {
		api.Route{
			Pattern: "/{id}",
			OpId:    "get_pack",
			Method:  api.GET,
			Docs:    get_pack.Docs,
			Handler: get_pack.Route,
		}.Route(r)

		api.Route{
			Pattern: "/all",
			OpId:    "get_all_packs",
			Method:  api.GET,
			Docs:    get_all_packs.Docs,
			Handler: get_all_packs.Route,
		}.Route(r)
	})
}
