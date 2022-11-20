package bots

import (
	"popplio/api"
	"popplio/routes/bots/endpoints/get_all_bots"
	"popplio/routes/bots/endpoints/get_bot"
	"popplio/routes/bots/endpoints/get_bot_reviews"
	"popplio/routes/bots/endpoints/get_bot_seo"
	"popplio/routes/bots/endpoints/post_stats"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Bots"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to bots on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/bots", func(r chi.Router) {
		api.Route{
			Pattern: "/all",
			Method:  api.GET,
			Docs:    get_all_bots.Docs,
			Handler: get_all_bots.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}",
			Method:  api.GET,
			Docs:    get_bot.Docs,
			Handler: get_bot.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/seo",
			Method:  api.GET,
			Docs:    get_bot_seo.Docs,
			Handler: get_bot_seo.Route,
		}.Route(r)

		api.Route{
			Pattern: "/stats",
			Method:  api.POST,
			Docs:    post_stats.Docs,
			Handler: post_stats.Route,
			Auth: []api.AuthType{
				{
					Type: types.TargetTypeBot,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reviews",
			Method:  api.GET,
			Docs:    get_bot_reviews.Docs,
			Handler: get_bot_reviews.Route,
		}.Route(r)
	})
}
