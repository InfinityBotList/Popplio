package list

import (
	"popplio/routes/list/endpoints/current_status"
	"popplio/routes/list/endpoints/get_cache_servers"
	"popplio/routes/list/endpoints/get_list_stats"
	"popplio/routes/list/endpoints/get_list_team"
	"popplio/routes/list/endpoints/get_partners"
	"popplio/routes/list/endpoints/get_rss_feed"
	"popplio/routes/list/endpoints/get_sitemap"
	"popplio/routes/list/endpoints/get_staff_templates"
	"popplio/routes/list/endpoints/search_list"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "List"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are core endpoints of our list."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/list/rss.xml",
		OpId:    "get_rss_feed",
		Method:  uapi.GET,
		Docs:    get_rss_feed.Docs,
		Handler: get_rss_feed.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/sitemap.xml",
		OpId:    "get_sitemap",
		Method:  uapi.GET,
		Docs:    get_sitemap.Docs,
		Handler: get_sitemap.Route,
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
		Pattern: "/list/partners",
		OpId:    "get_partners",
		Method:  uapi.GET,
		Docs:    get_partners.Docs,
		Handler: get_partners.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/list/staff-templates",
		OpId:    "get_partners",
		Method:  uapi.GET,
		Docs:    get_staff_templates.Docs,
		Handler: get_staff_templates.Route,
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

	uapi.Route{
		Pattern: "/list/cache-servers",
		OpId:    "get_cache_servers",
		Method:  uapi.GET,
		Docs:    get_cache_servers.Docs,
		Handler: get_cache_servers.Route,
	}.Route(r)
}
