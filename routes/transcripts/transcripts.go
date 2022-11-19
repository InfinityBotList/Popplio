package transcripts

import (
	"popplio/api"
	"popplio/routes/transcripts/endpoints/get_transcript"

	"github.com/go-chi/chi/v5"
)

const tagName = "Tickets + Transcripts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our ticketting and transcripts system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/transcripts", func(r chi.Router) {
		api.Route{
			Pattern: "/{id}",
			Method:  api.GET,
			Docs:    get_transcript.Docs,
			Handler: get_transcript.Route,
		}.Route(r)
	})
}
