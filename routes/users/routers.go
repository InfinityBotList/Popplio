package users

import (
	"popplio/api"
	"popplio/routes/users/endpoints/add_bot"
	"popplio/routes/users/endpoints/delete_user_notifications"
	"popplio/routes/users/endpoints/delete_user_reminders"
	"popplio/routes/users/endpoints/get_authorize_info"
	"popplio/routes/users/endpoints/get_notification_info"
	"popplio/routes/users/endpoints/get_user"
	"popplio/routes/users/endpoints/get_user_notifications"
	"popplio/routes/users/endpoints/get_user_reminders"
	"popplio/routes/users/endpoints/get_user_seo"
	"popplio/routes/users/endpoints/patch_bot_settings"
	"popplio/routes/users/endpoints/patch_bot_vanity"
	"popplio/routes/users/endpoints/patch_user_profile"
	"popplio/routes/users/endpoints/post_user_subscription"
	"popplio/routes/users/endpoints/put_user"
	"popplio/routes/users/endpoints/put_user_reminders"
	"popplio/types"

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
			OpId:    "get_user",
			Method:  api.GET,
			Docs:    get_user.Docs,
			Handler: get_user.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}",
			OpId:    "patch_user_profile",
			Method:  api.PATCH,
			Docs:    patch_user_profile.Docs,
			Handler: patch_user_profile.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/authorize",
			OpId:    "get_authorize_info",
			Method:  api.GET,
			Docs:    get_authorize_info.Docs,
			Handler: get_authorize_info.Route,
		}.Route(r)

		api.Route{
			Pattern: "/",
			OpId:    "put_user",
			Method:  api.PUT,
			Docs:    put_user.Docs,
			Handler: put_user.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/bots",
			OpId:    "add_bot",
			Method:  api.PUT,
			Docs:    add_bot.Docs,
			Handler: add_bot.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{uid}/bots/{bid}/vanity",
			OpId:    "patch_bot_vanity",
			Method:  api.PATCH,
			Docs:    patch_bot_vanity.Docs,
			Handler: patch_bot_vanity.Route,
			Auth: []api.AuthType{
				{
					URLVar: "uid",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{uid}/bots/{bid}/settings",
			OpId:    "patch_bot_settings",
			Method:  api.PATCH,
			Docs:    patch_bot_settings.Docs,
			Handler: patch_bot_settings.Route,
			Setup:   patch_bot_settings.Setup,
			Auth: []api.AuthType{
				{
					URLVar: "uid",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/seo",
			OpId:    "get_user_seo",
			Method:  api.GET,
			Docs:    get_user_seo.Docs,
			Handler: get_user_seo.Route,
		}.Route(r)

		api.Route{
			Pattern: "/notifications/info",
			OpId:    "get_notification_info",
			Method:  api.GET,
			Docs:    get_notification_info.Docs,
			Handler: get_notification_info.Route,
		}.Route(r)

		api.Route{
			Pattern: "/{id}/notifications",
			OpId:    "get_user_notifications",
			Method:  api.GET,
			Docs:    get_user_notifications.Docs,
			Handler: get_user_notifications.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/notification",
			OpId:    "delete_user_notifications",
			Method:  api.DELETE,
			Docs:    delete_user_notifications.Docs,
			Handler: delete_user_notifications.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reminders",
			OpId:    "get_user_reminders",
			Method:  api.GET,
			Docs:    get_user_reminders.Docs,
			Handler: get_user_reminders.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reminders",
			OpId:    "put_user_reminders",
			Method:  api.PUT,
			Docs:    put_user_reminders.Docs,
			Handler: put_user_reminders.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/reminder",
			OpId:    "delete_user_reminders",
			Method:  api.DELETE,
			Docs:    delete_user_reminders.Docs,
			Handler: delete_user_reminders.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)

		api.Route{
			Pattern: "/{id}/sub",
			OpId:    "post_user_subscription",
			Method:  api.POST,
			Docs:    post_user_subscription.Docs,
			Handler: post_user_subscription.Route,
			Auth: []api.AuthType{
				{
					URLVar: "id",
					Type:   types.TargetTypeUser,
				},
			},
		}.Route(r)
	})
}
