package list

import (
	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/get_list_index"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/get_list_stats"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/get_vote_info"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/parse_html"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/search_list"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/test_auth"
	"github.com/infinitybotlist/popplio/routes/list/endpoints/test_webhook"

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
			OpId:    "get_list_index",
			Method:  api.GET,
			Docs:    get_list_index.Docs,
			Handler: get_list_index.Route,
		}.Route(r)

		api.Route{
			Pattern: "/search",
			OpId:    "search_list",
			Method:  api.POST,
			Docs:    search_list.Docs,
			Handler: search_list.Route,
		}.Route(r)

		api.Route{
			Pattern: "/stats",
			OpId:    "get_list_stats",
			Method:  api.GET,
			Docs:    get_list_stats.Docs,
			Handler: get_list_stats.Route,
		}.Route(r)

		api.Route{
			Pattern: "/vote-info",
			OpId:    "get_vote_info",
			Method:  api.GET,
			Docs:    get_vote_info.Docs,
			Handler: get_vote_info.Route,
		}.Route(r)

		api.Route{
			Pattern: "/webhook-test",
			OpId:    "test_webhook",
			Method:  api.POST,
			Docs:    test_webhook.Docs,
			Handler: test_webhook.Route,
		}.Route(r)

		api.Route{
			Pattern: "/auth-test",
			OpId:    "test_auth",
			Method:  api.POST,
			Docs:    test_auth.Docs,
			Handler: test_auth.Route,
		}.Route(r)

		api.Route{
			Pattern: "/parse-html",
			OpId:    "parse_html",
			Method:  api.POST,
			Docs:    parse_html.Docs,
			Handler: parse_html.Route,
		}.Route(r)
	})
}
