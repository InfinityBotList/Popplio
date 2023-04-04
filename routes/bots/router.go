package bots

import (
	"popplio/api"
	"popplio/routes/bots/endpoints/add_bot"
	"popplio/routes/bots/endpoints/get_all_bots"
	"popplio/routes/bots/endpoints/get_bot"
	"popplio/routes/bots/endpoints/get_bot_invite"
	"popplio/routes/bots/endpoints/get_bot_meta"
	"popplio/routes/bots/endpoints/get_bot_seo"
	"popplio/routes/bots/endpoints/patch_bot_settings"
	"popplio/routes/bots/endpoints/patch_bot_vanity"
	"popplio/routes/bots/endpoints/patch_bot_webhook"
	"popplio/routes/bots/endpoints/post_stats"
	"popplio/routes/bots/endpoints/reset_bot_token"

	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Bots"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to bots on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/bots/all",
		OpId:    "get_all_bots",
		Method:  api.GET,
		Docs:    get_all_bots.Docs,
		Handler: get_all_bots.Route,
	}.Route(r)

	api.Route{
		Pattern: "/bots/{id}",
		OpId:    "get_bot",
		Method:  api.GET,
		Docs:    get_bot.Docs,
		Handler: get_bot.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{cid}/meta",
		OpId:    "get_bot_meta",
		Method:  api.GET,
		Docs:    get_bot_meta.Docs,
		Handler: get_bot_meta.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/bots/{id}/invite",
		OpId:    "get_bot_invite",
		Method:  api.GET,
		Docs:    get_bot_invite.Docs,
		Handler: get_bot_invite.Route,
	}.Route(r)

	api.Route{
		Pattern: "/bots/{id}/seo",
		OpId:    "get_bot_seo",
		Method:  api.GET,
		Docs:    get_bot_seo.Docs,
		Handler: get_bot_seo.Route,
	}.Route(r)

	api.Route{
		Pattern: "/bots/stats",
		OpId:    "post_stats",
		Method:  api.POST,
		Docs:    post_stats.Docs,
		Handler: post_stats.Route,
		Auth: []api.AuthType{
			{
				Type: api.TargetTypeBot,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/bots",
		OpId:    "add_bot",
		Method:  api.PUT,
		Docs:    add_bot.Docs,
		Handler: add_bot.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
		Setup: add_bot.Setup,
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/settings",
		OpId:    "patch_bot_settings",
		Method:  api.PATCH,
		Docs:    patch_bot_settings.Docs,
		Handler: patch_bot_settings.Route,
		Setup:   patch_bot_settings.Setup,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/vanity",
		OpId:    "patch_bot_vanity",
		Method:  api.PATCH,
		Docs:    patch_bot_vanity.Docs,
		Handler: patch_bot_vanity.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/webhook",
		OpId:    "patch_bot_webhook",
		Method:  api.PATCH,
		Docs:    patch_bot_webhook.Docs,
		Handler: patch_bot_webhook.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/token",
		OpId:    "reset_bot_token",
		Method:  api.PATCH,
		Docs:    reset_bot_token.Docs,
		Handler: reset_bot_token.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
