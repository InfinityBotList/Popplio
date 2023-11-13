package users

import (
	"popplio/api"
	"popplio/routes/users/endpoints/check_booster_status"
	"popplio/routes/users/endpoints/create_data_task"
	"popplio/routes/users/endpoints/get_user"
	"popplio/routes/users/endpoints/get_user_perms"
	"popplio/routes/users/endpoints/get_user_seo"
	"popplio/routes/users/endpoints/patch_user_profile"
	"popplio/routes/users/endpoints/put_user"
	"popplio/routes/users/endpoints/reset_user_token"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Users"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to users on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/users/{id}/booster",
		OpId:    "check_booster_status",
		Method:  uapi.GET,
		Docs:    check_booster_status.Docs,
		Handler: check_booster_status.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/data",
		OpId:    "create_data_task",
		Method:  uapi.POST,
		Docs:    create_data_task.Docs,
		Handler: create_data_task.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}",
		OpId:    "get_user",
		Method:  uapi.GET,
		Docs:    get_user.Docs,
		Handler: get_user.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/perms",
		OpId:    "get_user_perms",
		Method:  uapi.GET,
		Docs:    get_user_perms.Docs,
		Handler: get_user_perms.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/seo",
		OpId:    "get_user_seo",
		Method:  uapi.GET,
		Docs:    get_user_seo.Docs,
		Handler: get_user_seo.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}",
		OpId:    "patch_user_profile",
		Method:  uapi.PATCH,
		Docs:    patch_user_profile.Docs,
		Handler: patch_user_profile.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users",
		OpId:    "put_user",
		Method:  uapi.PUT,
		Docs:    put_user.Docs,
		Handler: put_user.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/token",
		OpId:    "reset_user_token",
		Method:  uapi.PATCH,
		Docs:    reset_user_token.Docs,
		Handler: reset_user_token.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
