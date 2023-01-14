package reviews

import (
	"popplio/api"
	"popplio/routes/reviews/endpoints/get_bot_reviews"

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
}
