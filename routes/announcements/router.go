package announcements

import (
	"popplio/api"
	"popplio/routes/announcements/endpoints/get_announcements"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const tagName = "Announcements"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our announcements system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/announcements", func(r chi.Router) {
		api.Route{
			Pattern: "/",
			OpId:    "get_announcements",
			Method:  api.GET,
			Docs:    get_announcements.Docs,
			Handler: get_announcements.Route,
			Auth: []api.AuthType{
				{
					Type: types.TargetTypeUser,
				},
			},
			AuthOptional: true,
		}.Route(r)
	})
}
