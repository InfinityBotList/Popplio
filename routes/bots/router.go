package bots

import (
	"popplio/api"
	"popplio/routes/bots/endpoints/add_bot"
	"popplio/routes/bots/endpoints/delete_bot"
	"popplio/routes/bots/endpoints/get_all_bots"
	"popplio/routes/bots/endpoints/get_bot"
	"popplio/routes/bots/endpoints/get_bot_meta"
	"popplio/routes/bots/endpoints/get_bot_seo"
	"popplio/routes/bots/endpoints/get_random_bots"
	"popplio/routes/bots/endpoints/patch_bot_settings"
	"popplio/routes/bots/endpoints/patch_bot_team"
	"popplio/routes/bots/endpoints/post_bot_stats"
	"popplio/routes/bots/endpoints/transfer_bot_to_team"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
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
		Pattern: "/bots/{id}",
		OpId:    "get_bot",
		Method:  uapi.GET,
		Docs:    get_bot.Docs,
		Handler: get_bot.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{cid}/meta",
		OpId:    "get_bot_meta",
		Method:  uapi.GET,
		Docs:    get_bot_meta.Docs,
		Handler: get_bot_meta.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
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
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/bots",
		OpId:    "add_bot",
		Method:  uapi.PUT,
		Docs:    add_bot.Docs,
		Handler: add_bot.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
		Setup: add_bot.Setup,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}",
		OpId:    "delete_bot",
		Method:  uapi.DELETE,
		Docs:    delete_bot.Docs,
		Handler: delete_bot.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/settings",
		OpId:    "patch_bot_settings",
		Method:  uapi.PATCH,
		Docs:    patch_bot_settings.Docs,
		Handler: patch_bot_settings.Route,
		Setup:   patch_bot_settings.Setup,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/teams",
		OpId:    "transfer_bot_to_team",
		Method:  uapi.PUT,
		Docs:    transfer_bot_to_team.Docs,
		Handler: transfer_bot_to_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
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
	}.Route(r)
}
