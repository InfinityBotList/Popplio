package auth

import (
	"popplio/api"
	"popplio/routes/auth/endpoints/create_oauth2_login"
	"popplio/routes/auth/endpoints/create_session"
	"popplio/routes/auth/endpoints/get_oauth_url"
	"popplio/routes/auth/endpoints/get_sessions"
	"popplio/routes/auth/endpoints/revoke_session"
	"popplio/routes/auth/endpoints/test_auth"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "API Tokens"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to API Tokens on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/{target_type}/{target_id}/sessions",
		OpId:    "get_sessions",
		Method:  uapi.GET,
		Docs:    get_sessions.Docs,
		Handler: get_sessions.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeBot,
			},
			{
				Type: api.TargetTypeServer,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/sessions/{target_type}/{target_id}",
		OpId:    "create_session",
		Method:  uapi.POST,
		Docs:    create_session.Docs,
		Handler: create_session.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeBot,
			},
			{
				Type: api.TargetTypeServer,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/sessions/{target_type}/{target_id}",
		OpId:    "revoke_session",
		Method:  uapi.DELETE,
		Docs:    revoke_session.Docs,
		Handler: revoke_session.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeBot,
			},
			{
				Type: api.TargetTypeServer,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/auth/login/discord-oauth2",
		OpId:    "get_oauth_url",
		Method:  uapi.GET,
		Docs:    get_oauth_url.Docs,
		Handler: get_oauth_url.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/auth/login/discord-oauth2",
		OpId:    "create_oauth2_login",
		Method:  uapi.POST,
		Docs:    create_oauth2_login.Docs,
		Handler: create_oauth2_login.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/auth/test",
		OpId:    "test_auth",
		Method:  uapi.POST,
		Docs:    test_auth.Docs,
		Handler: test_auth.Route,
	}.Route(r)
}
