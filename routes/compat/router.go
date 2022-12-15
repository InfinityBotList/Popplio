package compat

import (
	"popplio/api"
	"popplio/routes/compat/endpoints/legacy_votes"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const tagName = "Legacy"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are set to be removed in the next major API version. Please use the new endpoints instead."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/votes/{bot_id}/{user_id}",
		OpId:    "legacy_votes",
		Method:  api.GET,
		Docs:    legacy_votes.Docs,
		Handler: legacy_votes.Route,
		Auth: []api.AuthType{
			{
				URLVar: "bot_id",
				Type:   types.TargetTypeBot,
			},
		},
	}.Route(r)
}
