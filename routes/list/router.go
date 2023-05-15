package list

import (
	"popplio/routes/list/endpoints/current_status"
	"popplio/routes/list/endpoints/get_list_index"
	"popplio/routes/list/endpoints/get_list_stats"
	"popplio/routes/list/endpoints/get_list_team"
	"popplio/routes/list/endpoints/get_partners"
	"popplio/routes/list/endpoints/search_list"
	"popplio/routes/list/endpoints/test_auth"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "List Stats"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are basic statistics of our list."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/list/index",
		OpId:    "get_list_index",
		Method:  uapi.GET,
		Docs:    get_list_index.Docs,
		Handler: get_list_index.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/search",
		OpId:    "search_list",
		Method:  uapi.POST,
		Docs:    search_list.Docs,
		Handler: search_list.Route,
		Setup:   search_list.Setup,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/stats",
		OpId:    "get_list_stats",
		Method:  uapi.GET,
		Docs:    get_list_stats.Docs,
		Handler: get_list_stats.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/auth-test",
		OpId:    "test_auth",
		Method:  uapi.POST,
		Docs:    test_auth.Docs,
		Handler: test_auth.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/partners",
		OpId:    "get_partners",
		Method:  uapi.GET,
		Docs:    get_partners.Docs,
		Handler: get_partners.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/team",
		OpId:    "get_list_team",
		Method:  uapi.GET,
		Docs:    get_list_team.Docs,
		Handler: get_list_team.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/current-status",
		OpId:    "current_status",
		Method:  uapi.GET,
		Docs:    current_status.Docs,
		Handler: current_status.Route,
	}.Route(r)
}
