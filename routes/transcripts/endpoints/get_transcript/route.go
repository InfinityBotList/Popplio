package get_transcript

import (
	"net/http"
	"strconv"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/state"
	"github.com/infinitybotlist/popplio/types"

	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	transcriptNum := chi.URLParam(r, "id")

	if transcriptNum == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	transcriptNumInt, err := strconv.Atoi(transcriptNum)

	if err != nil {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Get transcript
	var data []byte
	var closedBy []byte
	var openedBy []byte

	err = state.Pool.QueryRow(d.Context, "SELECT data, closed_by, opened_by FROM transcripts WHERE id = $1", transcriptNumInt).Scan(&data, &closedBy, &openedBy)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	var dataParsed []map[string]any

	err = json.Unmarshal(data, &dataParsed)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var closedByParsed map[string]any

	err = json.Unmarshal(closedBy, &closedByParsed)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var openedByParsed map[string]any

	err = json.Unmarshal(openedBy, &openedByParsed)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var transcript = types.Transcript{
		ID:       transcriptNumInt,
		Data:     dataParsed,
		ClosedBy: closedByParsed,
		OpenedBy: openedByParsed,
	}

	return api.HttpResponse{
		Json: transcript,
	}
}
