package blogs

import (
	"popplio/api"
	"popplio/routes/blogs/endpoints/create_blog_post"
	"popplio/routes/blogs/endpoints/delete_blog_post"
	"popplio/routes/blogs/endpoints/get_blog_list"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const tagName = "Blog"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to blog posts on our list."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/users/{user_id}/blog",
		OpId:    "create_blog_post",
		Method:  api.POST,
		Docs:    create_blog_post.Docs,
		Handler: create_blog_post.Route,
		Auth: []api.AuthType{
			{
				URLVar: "user_id",
				Type:   types.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{user_id}/blog",
		OpId:    "delete_blog_post",
		Method:  api.DELETE,
		Docs:    delete_blog_post.Docs,
		Handler: delete_blog_post.Route,
		Auth: []api.AuthType{
			{
				URLVar: "user_id",
				Type:   types.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/blog",
		OpId:    "get_blog_list",
		Method:  api.GET,
		Docs:    get_blog_list.Docs,
		Handler: get_blog_list.Route,
	}.Route(r)
}
