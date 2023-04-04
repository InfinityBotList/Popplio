package users

import (
	"popplio/api"
	"popplio/routes/users/endpoints/create_data_task"
	"popplio/routes/users/endpoints/delete_user_notifications"
	"popplio/routes/users/endpoints/delete_user_reminders"
	"popplio/routes/users/endpoints/get_data_task"
	"popplio/routes/users/endpoints/get_notification_info"
	"popplio/routes/users/endpoints/get_user"
	"popplio/routes/users/endpoints/get_user_alerts"
	"popplio/routes/users/endpoints/get_user_notifications"
	"popplio/routes/users/endpoints/get_user_perms"
	"popplio/routes/users/endpoints/get_user_reminders"
	"popplio/routes/users/endpoints/get_user_seo"
	"popplio/routes/users/endpoints/patch_user_profile"
	"popplio/routes/users/endpoints/post_user_subscription"
	"popplio/routes/users/endpoints/put_user"
	"popplio/routes/users/endpoints/put_user_reminders"
	"popplio/routes/users/endpoints/reset_user_token"

	"github.com/go-chi/chi/v5"
)

const tagName = "Users"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to users on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/users/{id}",
		OpId:    "get_user",
		Method:  api.GET,
		Docs:    get_user.Docs,
		Handler: get_user.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}",
		OpId:    "patch_user_profile",
		Method:  api.PATCH,
		Docs:    patch_user_profile.Docs,
		Handler: patch_user_profile.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/token",
		OpId:    "reset_user_token",
		Method:  api.PATCH,
		Docs:    reset_user_token.Docs,
		Handler: reset_user_token.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/data/{tid}",
		OpId:    "get_data_task",
		Method:  api.GET,
		Docs:    get_data_task.Docs,
		Handler: get_data_task.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/data",
		OpId:    "create_data_task",
		Method:  api.POST,
		Docs:    create_data_task.Docs,
		Handler: create_data_task.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users",
		OpId:    "put_user",
		Method:  api.PUT,
		Docs:    put_user.Docs,
		Handler: put_user.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/seo",
		OpId:    "get_user_seo",
		Method:  api.GET,
		Docs:    get_user_seo.Docs,
		Handler: get_user_seo.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/perms",
		OpId:    "get_user_perms",
		Method:  api.GET,
		Docs:    get_user_perms.Docs,
		Handler: get_user_perms.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/notifications/info",
		OpId:    "get_notification_info",
		Method:  api.GET,
		Docs:    get_notification_info.Docs,
		Handler: get_notification_info.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/notifications",
		OpId:    "get_user_notifications",
		Method:  api.GET,
		Docs:    get_user_notifications.Docs,
		Handler: get_user_notifications.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/notification",
		OpId:    "delete_user_notifications",
		Method:  api.DELETE,
		Docs:    delete_user_notifications.Docs,
		Handler: delete_user_notifications.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/alerts",
		OpId:    "get_user_alerts",
		Method:  api.GET,
		Docs:    get_user_alerts.Docs,
		Handler: get_user_alerts.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/reminders",
		OpId:    "get_user_reminders",
		Method:  api.GET,
		Docs:    get_user_reminders.Docs,
		Handler: get_user_reminders.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/reminders/{bid}",
		OpId:    "put_user_reminders",
		Method:  api.PUT,
		Docs:    put_user_reminders.Docs,
		Handler: put_user_reminders.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/reminders/{bid}",
		OpId:    "delete_user_reminders",
		Method:  api.DELETE,
		Docs:    delete_user_reminders.Docs,
		Handler: delete_user_reminders.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/sub",
		OpId:    "post_user_subscription",
		Method:  api.POST,
		Docs:    post_user_subscription.Docs,
		Handler: post_user_subscription.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
