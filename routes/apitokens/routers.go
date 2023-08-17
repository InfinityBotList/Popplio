package apitokens

import (
	"popplio/api"
	"popplio/routes/apitokens/endpoints/get_entity_token"
	"popplio/routes/apitokens/endpoints/reset_entity_token"

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
		Pattern: "/users/{uid}/tokens/{target_id}",
		OpId:    "get_entity_token",
		Method:  uapi.GET,
		Docs:    get_entity_token.Docs,
		Handler: get_entity_token.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/tokens/{target_id}",
		OpId:    "reset_entity_token",
		Method:  uapi.PATCH,
		Docs:    reset_entity_token.Docs,
		Handler: reset_entity_token.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
