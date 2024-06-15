package notifications

import (
	"popplio/api"
	"popplio/routes/notifications/endpoints/create_user_notifications"
	"popplio/routes/notifications/endpoints/delete_user_notifications"
	"popplio/routes/notifications/endpoints/get_notification_info"
	"popplio/routes/notifications/endpoints/get_user_notifications"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Notifications"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to user notifications on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/users/notifications/info",
		OpId:    "get_notification_info",
		Method:  uapi.GET,
		Docs:    get_notification_info.Docs,
		Handler: get_notification_info.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/notifications",
		OpId:    "get_user_notifications",
		Method:  uapi.GET,
		Docs:    get_user_notifications.Docs,
		Handler: get_user_notifications.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/notification",
		OpId:    "delete_user_notifications",
		Method:  uapi.DELETE,
		Docs:    delete_user_notifications.Docs,
		Handler: delete_user_notifications.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/notifications",
		OpId:    "create_user_notifications",
		Method:  uapi.POST,
		Docs:    create_user_notifications.Docs,
		Handler: create_user_notifications.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
