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
	jsoniter "github.com/json-iterator/go"
)

const tagName = "Tickets + Transcripts"

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

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
			transcriptNum := chi.URLParam(r, "id")

			if transcriptNum == "" {
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			transcriptNumInt, err := strconv.Atoi(transcriptNum)

			if err != nil {
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			// Get transcript
			var data pgtype.JSONB
			var closedBy pgtype.JSONB
			var openedBy pgtype.JSONB

			err = state.Pool.QueryRow(state.Context, "SELECT data, closed_by, opened_by FROM transcripts WHERE id = $1", transcriptNumInt).Scan(&data, &closedBy, &openedBy)

			if err != nil {
				state.Logger.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			var transcript = types.Transcript{
				ID:       transcriptNumInt,
				Data:     data,
				ClosedBy: closedBy,
				OpenedBy: openedBy,
			}

			bytes, err := json.Marshal(transcript)

			if err != nil {
				state.Logger.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		})
	})
}
