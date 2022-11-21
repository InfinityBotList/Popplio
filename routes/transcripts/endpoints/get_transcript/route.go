package get_transcript

import (
	"encoding/json"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/transcripts/{id}",
		OpId:        "get_transcript",
		Summary:     "Get Ticket Transcript",
		Description: "Gets the transcript of a ticket. **Note that this endpoint is only documented to be useful for staff and the like. It is not useful for normal users**",
		Tags:        []string{api.CurrentTag},
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
}

func Route(d api.RouteData, r *http.Request) {
	transcriptNum := chi.URLParam(r, "id")

	if transcriptNum == "" {
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	transcriptNumInt, err := strconv.Atoi(transcriptNum)

	if err != nil {
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	// Get transcript
	var data []byte
	var closedBy []byte
	var openedBy []byte

	err = state.Pool.QueryRow(d.Context, "SELECT data, closed_by, opened_by FROM transcripts WHERE id = $1", transcriptNumInt).Scan(&data, &closedBy, &openedBy)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	var dataParsed []map[string]any

	err = json.Unmarshal(data, &dataParsed)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var closedByParsed map[string]any

	err = json.Unmarshal(closedBy, &closedByParsed)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var openedByParsed map[string]any

	err = json.Unmarshal(openedBy, &openedByParsed)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var transcript = types.Transcript{
		ID:       transcriptNumInt,
		Data:     dataParsed,
		ClosedBy: closedByParsed,
		OpenedBy: openedByParsed,
	}

	d.Resp <- api.HttpResponse{
		Json: transcript,
	}
}
