package transcripts

import (
	"net/http"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgtype"
)

const tagName = "Tickets + Transcripts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our ticketting and transcripts system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/transcripts", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/transcripts/{id}",
			OpId:        "get_transcript",
			Summary:     "Get Ticket Transcript",
			Description: "Gets the transcript of a ticket. **Note that this endpoint is only documented to be useful for staff and the like. It is not useful for normal users**",
			Tags:        []string{tagName},
			Params: []docs.Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "The ticket's ID",
					Required:    true,
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.Transcript{},
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				transcriptNum := chi.URLParam(r, "id")

				if transcriptNum == "" {
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				transcriptNumInt, err := strconv.Atoi(transcriptNum)

				if err != nil {
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				// Get transcript
				var data pgtype.JSONB
				var closedBy pgtype.JSONB
				var openedBy pgtype.JSONB

				err = state.Pool.QueryRow(ctx, "SELECT data, closed_by, opened_by FROM transcripts WHERE id = $1", transcriptNumInt).Scan(&data, &closedBy, &openedBy)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				var transcript = types.Transcript{
					ID:       transcriptNumInt,
					Data:     data,
					ClosedBy: closedBy,
					OpenedBy: openedBy,
				}

				resp <- types.HttpResponse{
					Json: transcript,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}