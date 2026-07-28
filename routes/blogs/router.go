package blogs

import (
	"popplio/routes/blogs/endpoints/get_blog_list"
	"popplio/routes/blogs/endpoints/get_blog_post"
	"popplio/routes/blogs/endpoints/get_blog_seo"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Blog"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to blog posts on our list."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/blogs/@all",
		OpId:    "get_blog_list",
		Method:  uapi.GET,
		Docs:    get_blog_list.Docs,
		Handler: get_blog_list.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/blogs/{slug}",
		OpId:    "get_blog_post",
		Method:  uapi.GET,
		Docs:    get_blog_post.Docs,
		Handler: get_blog_post.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/blogs/{slug}/seo",
		OpId:    "get_blog_seo",
		Method:  uapi.GET,
		Docs:    get_blog_seo.Docs,
		Handler: get_blog_seo.Route,
	}.Route(r)
}
