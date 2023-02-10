package list

import (
	"popplio/api"
	"popplio/routes/list/endpoints/get_list_index"
	"popplio/routes/list/endpoints/get_list_stats"
	"popplio/routes/list/endpoints/get_partners"
	"popplio/routes/list/endpoints/get_random_bots"
	"popplio/routes/list/endpoints/parse_html"
	"popplio/routes/list/endpoints/search_list"
	"popplio/routes/list/endpoints/test_auth"

	"github.com/go-chi/chi/v5"
)

const tagName = "List Stats"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are basic statistics of our list."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/list/index",
		OpId:    "get_list_index",
		Method:  api.GET,
		Docs:    get_list_index.Docs,
		Handler: get_list_index.Route,
	}.Route(r)

	api.Route{
		Pattern: "/list/random-bots",
		OpId:    "get_random_bots",
		Method:  api.GET,
		Docs:    get_random_bots.Docs,
		Handler: get_random_bots.Route,
	}.Route(r)

	api.Route{
		Pattern: "/list/search",
		OpId:    "search_list",
		Method:  api.POST,
		Docs:    search_list.Docs,
		Handler: search_list.Route,
		Setup:   search_list.Setup,
	}.Route(r)

	api.Route{
		Pattern: "/list/stats",
		OpId:    "get_list_stats",
		Method:  api.GET,
		Docs:    get_list_stats.Docs,
		Handler: get_list_stats.Route,
	}.Route(r)

	api.Route{
		Pattern: "/list/auth-test",
		OpId:    "test_auth",
		Method:  api.POST,
		Docs:    test_auth.Docs,
		Handler: test_auth.Route,
	}.Route(r)

	api.Route{
		Pattern: "/list/parse-html",
		OpId:    "parse_html",
		Method:  api.POST,
		Docs:    parse_html.Docs,
		Handler: parse_html.Route,
		Setup:   parse_html.Setup,
	}.Route(r)

	api.Route{
		Pattern: "/list/partners",
		OpId:    "get_partners",
		Method:  api.GET,
		Docs:    get_partners.Docs,
		Handler: get_partners.Route,
	}.Route(r)
}
