package reviews

import (
	"popplio/api"
	"popplio/routes/reviews/endpoints/add_bot_review"
	"popplio/routes/reviews/endpoints/get_bot_reviews"
	"popplio/routes/reviews/endpoints/remove_bot_review"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Reviews"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to reviews on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/bots/{id}/reviews",
		OpId:    "get_bot_reviews",
		Method:  api.GET,
		Docs:    get_bot_reviews.Docs,
		Handler: get_bot_reviews.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/reviews",
		OpId:    "add_bot_review",
		Method:  api.POST,
		Docs:    add_bot_review.Docs,
		Handler: add_bot_review.Route,
		Auth: []api.AuthType{
			{
				Type:   types.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/reviews/{rid}",
		OpId:    "remove_bot_review",
		Method:  api.DELETE,
		Docs:    remove_bot_review.Docs,
		Handler: remove_bot_review.Route,
		Auth: []api.AuthType{
			{
				Type:   types.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
