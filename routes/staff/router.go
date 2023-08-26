package staff

import (
	"popplio/routes/staff/endpoints/create_blog_post"
	"popplio/routes/staff/endpoints/delete_blog_post"
	"popplio/routes/staff/endpoints/edit_blog_post"
	"popplio/routes/staff/endpoints/manage_app"
	"popplio/routes/staff/endpoints/publish_blog_post"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const (
	tagName = "Staff"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Staff-only IBL endpoints. Only usable from staff panel using panelapi credentials"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/staff/@manage/blog",
		OpId:    "create_blog_post",
		Method:  uapi.POST,
		Docs:    create_blog_post.Docs,
		Handler: create_blog_post.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/@manage/blog/{slug}",
		OpId:    "edit_blog_post",
		Method:  uapi.PATCH,
		Docs:    edit_blog_post.Docs,
		Handler: edit_blog_post.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/@manage/{slug}/draft",
		OpId:    "publish_blog_post",
		Method:  uapi.PATCH,
		Docs:    publish_blog_post.Docs,
		Handler: publish_blog_post.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/@manage/blog/{slug}",
		OpId:    "delete_blog_post",
		Method:  uapi.DELETE,
		Docs:    delete_blog_post.Docs,
		Handler: delete_blog_post.Route,
	}.Route(r)
	uapi.Route{
		Pattern: "/staff/@manage/apps/{app_id}",
		OpId:    "manage_app",
		Method:  uapi.PATCH,
		Docs:    manage_app.Docs,
		Handler: manage_app.Route,
	}.Route(r)
}
