package put_user_pack_votes

import (
	"encoding/json"
	"io"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"

	"github.com/go-chi/chi/v5"
)

type CreatePackVote struct {
	Upvote bool `json:"upvote"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PUT",
		Path:        "/users/{uid}/packs/{url}/votes",
		OpId:        "put_pack_vote",
		Summary:     "Create Pack Vote",
		Description: "Vote on a pack. You can vote for a pack only once but can change between upvote and downvote or use ``De;ete Pack Vote`` to delete their vote.",
		Tags:        []string{api.CurrentTag},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The ID of the pack.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req: CreatePackVote{},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var userId = d.Auth.ID
	var packUrl = chi.URLParam(r, "url")

	var vote CreatePackVote

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bodyBytes, &vote)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", packUrl).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if count == 0 {
		d.Resp <- api.DefaultResponse(http.StatusNotFound)
		return
	}

	// Check if the user has already voted and if so update the vote
	var packVotes int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM pack_votes WHERE user_id = $1 AND url = $2", userId, packUrl).Scan(&packVotes)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if packVotes == 0 {
		// We can freely insert a new vote
		_, err = state.Pool.Exec(d.Context, "INSERT INTO pack_votes (user_id, url, upvote) VALUES ($1, $2, $3)", userId, packUrl, vote.Upvote)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
			return
		}
	} else {
		// Update the vote
		_, err = state.Pool.Exec(d.Context, "UPDATE pack_votes SET upvote = $1 WHERE user_id = $2 AND url = $3", vote.Upvote, userId, packUrl)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
			return
		}
	}
}
