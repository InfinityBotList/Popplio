package blogs

import (
	"popplio/api"
	"popplio/routes/blogs/endpoints/create_blog_post"
	"popplio/routes/blogs/endpoints/delete_blog_post"
	"popplio/routes/blogs/endpoints/edit_blog_post"
	"popplio/routes/blogs/endpoints/get_blog_list"
	"popplio/routes/blogs/endpoints/get_blog_post"
	"popplio/routes/blogs/endpoints/get_blog_seo"
	"popplio/routes/blogs/endpoints/publish_blog_post"

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
		Pattern: "/users/{user_id}/blog",
		OpId:    "create_blog_post",
		Method:  uapi.POST,
		Docs:    create_blog_post.Docs,
		Handler: create_blog_post.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "user_id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{user_id}/blog/{slug}",
		OpId:    "edit_blog_post",
		Method:  uapi.PATCH,
		Docs:    edit_blog_post.Docs,
		Handler: edit_blog_post.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "user_id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{user_id}/blog/{slug}/draft",
		OpId:    "publish_blog_post",
		Method:  uapi.PATCH,
		Docs:    publish_blog_post.Docs,
		Handler: publish_blog_post.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "user_id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{user_id}/blog/{slug}",
		OpId:    "delete_blog_post",
		Method:  uapi.DELETE,
		Docs:    delete_blog_post.Docs,
		Handler: delete_blog_post.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "user_id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/blog",
		OpId:    "get_blog_list",
		Method:  uapi.GET,
		Docs:    get_blog_list.Docs,
		Handler: get_blog_list.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/blog/{slug}",
		OpId:    "get_blog_post",
		Method:  uapi.GET,
		Docs:    get_blog_post.Docs,
		Handler: get_blog_post.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/blog/{slug}/seo",
		OpId:    "get_blog_seo",
		Method:  uapi.GET,
		Docs:    get_blog_seo.Docs,
		Handler: get_blog_seo.Route,
	}.Route(r)
}
