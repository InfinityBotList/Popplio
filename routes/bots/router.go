package bots

import (
	"net/http"
	"popplio/api"
	"popplio/routes/bots/endpoints/add_bot"
	"popplio/routes/bots/endpoints/delete_bot"
	"popplio/routes/bots/endpoints/get_all_bots"
	"popplio/routes/bots/endpoints/get_bot"
	"popplio/routes/bots/endpoints/get_bot_meta"
	"popplio/routes/bots/endpoints/get_bot_seo"
	"popplio/routes/bots/endpoints/get_bots_index"
	"popplio/routes/bots/endpoints/get_random_bots"
	"popplio/routes/bots/endpoints/patch_bot_settings"
	"popplio/routes/bots/endpoints/patch_bot_team"
	"popplio/routes/bots/endpoints/post_bot_stats"
	"popplio/teams"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
)

const (
	tagName = "Bots"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to bots on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/bots/@all",
		OpId:    "get_all_bots",
		Method:  uapi.GET,
		Docs:    get_all_bots.Docs,
		Handler: get_all_bots.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/@random",
		OpId:    "get_random_bots",
		Method:  uapi.GET,
		Docs:    get_random_bots.Docs,
		Handler: get_random_bots.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/@index",
		OpId:    "get_bots_index",
		Method:  uapi.GET,
		Docs:    get_bots_index.Docs,
		Handler: get_bots_index.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/{id}",
		OpId:    "get_bot",
		Method:  uapi.GET,
		Docs:    get_bot.Docs,
		Handler: get_bot.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/{client_id}/meta",
		OpId:    "get_bot_meta",
		Method:  uapi.GET,
		Docs:    get_bot_meta.Docs,
		Handler: get_bot_meta.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: api.TargetTypeBot,
						Perm:      teams.PermissionAdd,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeBot, chi.URLParam(r, "client_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/{id}/seo",
		OpId:    "get_bot_seo",
		Method:  uapi.GET,
		Docs:    get_bot_seo.Docs,
		Handler: get_bot_seo.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/stats",
		OpId:    "post_bot_stats",
		Method:  uapi.POST,
		Docs:    post_bot_stats.Docs,
		Handler: post_bot_stats.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeBot,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/bots",
		OpId:    "add_bot",
		Method:  uapi.PUT,
		Docs:    add_bot.Docs,
		Handler: add_bot.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		Setup: add_bot.Setup,
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // The endpoint itself handles authorization
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/{id}",
		OpId:    "delete_bot",
		Method:  uapi.DELETE,
		Docs:    delete_bot.Docs,
		Handler: delete_bot.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: api.TargetTypeBot,
						Perm:      teams.PermissionDelete,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeBot, chi.URLParam(r, "id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/{id}/settings",
		OpId:    "patch_bot_settings",
		Method:  uapi.PATCH,
		Docs:    patch_bot_settings.Docs,
		Handler: patch_bot_settings.Route,
		Setup:   patch_bot_settings.Setup,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
			{
				Type: api.TargetTypeBot,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: api.TargetTypeBot,
						Perm:      teams.PermissionEdit,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeBot, chi.URLParam(r, "id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/teams",
		OpId:    "patch_bot_team",
		Method:  uapi.PATCH,
		Docs:    patch_bot_team.Docs,
		Handler: patch_bot_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/teams",
		OpId:    "patch_bot_team",
		Method:  uapi.PATCH,
		Docs:    patch_bot_team.Docs,
		Handler: patch_bot_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
