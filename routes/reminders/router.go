package reminders

import (
	"popplio/api"
	"popplio/routes/reminders/endpoints/delete_user_reminders"
	"popplio/routes/reminders/endpoints/get_user_reminders"
	"popplio/routes/reminders/endpoints/put_user_reminders"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Reminders"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to reminders on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/users/{id}/reminders",
		OpId:    "get_user_reminders",
		Method:  uapi.GET,
		Docs:    get_user_reminders.Docs,
		Handler: get_user_reminders.Route,
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
		Pattern: "/users/{uid}/{target_type}/{target_id}/reminders",
		OpId:    "put_user_reminders",
		Method:  uapi.PUT,
		Docs:    put_user_reminders.Docs,
		Handler: put_user_reminders.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/{target_type}/{target_id}/reminders",
		OpId:    "delete_user_reminders",
		Method:  uapi.DELETE,
		Docs:    delete_user_reminders.Docs,
		Handler: delete_user_reminders.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
