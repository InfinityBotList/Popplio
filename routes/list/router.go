package list

import (
	"popplio/api"
	"popplio/routes/list/endpoints/get_list_index"
	"popplio/routes/list/endpoints/get_list_stats"
	"popplio/routes/list/endpoints/get_vote_info"
	"popplio/routes/list/endpoints/test_webhook"

	"github.com/go-chi/chi/v5"
)

const tagName = "List Stats"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are basic statistics of our list."
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/list", func(r chi.Router) {
		api.Route{
			Pattern: "/index",
			Method:  api.GET,
			Docs:    get_list_index.Docs,
			Handler: get_list_index.Route,
		}.Route(r)

		api.Route{
			Pattern: "/stats",
			Method:  api.GET,
			Docs:    get_list_stats.Docs,
			Handler: get_list_stats.Route,
		}.Route(r)

		api.Route{
			Pattern: "/vote-info",
			Method:  api.GET,
			Docs:    get_vote_info.Docs,
			Handler: get_vote_info.Route,
		}.Route(r)

		api.Route{
			Pattern: "/webhook-test",
			Method:  api.POST,
			Docs:    test_webhook.Docs,
			Handler: test_webhook.Route,
		}.Route(r)
	})
}
