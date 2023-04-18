package reviews

import (
	"popplio/api"
	"popplio/routes/reviews/endpoints/add_bot_review"
	"popplio/routes/reviews/endpoints/edit_review"
	"popplio/routes/reviews/endpoints/get_bot_reviews"
	"popplio/routes/reviews/endpoints/remove_review"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const (
	tagName = "Reviews"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to reviews on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/bots/{id}/reviews",
		OpId:    "get_bot_reviews",
		Method:  uapi.GET,
		Docs:    get_bot_reviews.Docs,
		Handler: get_bot_reviews.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/reviews",
		OpId:    "add_bot_review",
		Method:  uapi.POST,
		Docs:    add_bot_review.Docs,
		Handler: add_bot_review.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/reviews/{rid}",
		OpId:    "edit_review",
		Method:  uapi.PATCH,
		Docs:    edit_review.Docs,
		Handler: edit_review.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/reviews/{rid}",
		OpId:    "remove_review",
		Method:  uapi.DELETE,
		Docs:    remove_review.Docs,
		Handler: remove_review.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
