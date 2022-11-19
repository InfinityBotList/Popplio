package users

import (
	"popplio/api"
	"popplio/routes/users/endpoints/delete_user_notifications"
	"popplio/routes/users/endpoints/delete_user_reminders"
	"popplio/routes/users/endpoints/get_notification_info"
	"popplio/routes/users/endpoints/get_user"
	"popplio/routes/users/endpoints/get_user_notifications"
	"popplio/routes/users/endpoints/get_user_reminders"
	"popplio/routes/users/endpoints/get_user_seo"
	"popplio/routes/users/endpoints/get_user_votes"
	"popplio/routes/users/endpoints/patch_user_profile"
	"popplio/routes/users/endpoints/post_user_subscription"
	"popplio/routes/users/endpoints/put_user_reminders"
	"popplio/routes/users/endpoints/put_user_votes"

	"github.com/go-chi/chi/v5"
)

const tagName = "Users"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to users on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/users", func(r chi.Router) {
		api.Route{
			Pattern: "/{id}",
			Method:  api.GET,
			Docs:    get_user.Docs,
			Handler: get_user.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{uid}/bots/{bid}/votes",
			Method:  api.GET,
			Docs:    get_user_votes.Docs,
			Handler: get_user_votes.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{uid}/bots/{bid}/votes",
			Method:  api.PUT,
			Docs:    put_user_votes.Docs,
			Handler: put_user_votes.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/seo",
			Method:  api.GET,
			Docs:    get_user_seo.Docs,
			Handler: get_user_seo.Route,
		}.Route(r)

		api.Route{
			Pattern: "/notifications/info",
			Method:  api.GET,
			Docs:    get_notification_info.Docs,
			Handler: get_notification_info.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/notifications",
			Method:  api.GET,
			Docs:    get_user_notifications.Docs,
			Handler: get_user_notifications.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/notification",
			Method:  api.DELETE,
			Docs:    delete_user_notifications.Docs,
			Handler: delete_user_notifications.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reminders",
			Method:  api.GET,
			Docs:    get_user_reminders.Docs,
			Handler: get_user_reminders.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reminder",
			Method:  api.DELETE,
			Docs:    delete_user_reminders.Docs,
			Handler: delete_user_reminders.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reminders",
			Method:  api.PUT,
			Docs:    put_user_reminders.Docs,
			Handler: put_user_reminders.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/sub",
			Method:  api.POST,
			Docs:    post_user_subscription.Docs,
			Handler: post_user_subscription.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}",
			Method:  api.PATCH,
			Docs:    patch_user_profile.Docs,
			Handler: patch_user_profile.Route,
		}.Route(r)
	})
}
