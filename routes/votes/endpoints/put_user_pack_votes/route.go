package put_user_pack_votes

import (
	"net/http"

	"popplio/api"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

type CreatePackVote struct {
	Upvote bool `json:"upvote"`
	Clear  bool `json:"clear,omitempty"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Create User Pack Vote",
		Description: `Creates a vote for a pack. 

This updates any existing vote or creates a new one if none exist.  Returns 204 on success.`,
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "url",
				Description: "The pack URL",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  CreatePackVote{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var userId = d.Auth.ID

	var voteBannedState bool

	err := state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", userId).Scan(&voteBannedState)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBannedState {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You are banned from voting right now! Contact support if you think this is a mistake",
				Error:   true,
			},
		}
	}

	var packUrl = chi.URLParam(r, "url")

	var vote CreatePackVote

	var resp, ok = api.MarshalReq(r, &vote)

	if !ok {
		return resp
	}

	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", packUrl).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	if vote.Clear {
		_, err = state.Pool.Exec(d.Context, "DELETE FROM pack_votes WHERE user_id = $1 AND url = $2", userId, packUrl)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		return api.DefaultResponse(http.StatusNoContent)
	}

	// Check if the user has already voted and if so update the vote
	var packVotes int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM pack_votes WHERE user_id = $1 AND url = $2", userId, packUrl).Scan(&packVotes)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if packVotes == 0 {
		// We can freely insert a new vote
		_, err = state.Pool.Exec(d.Context, "INSERT INTO pack_votes (user_id, url, upvote) VALUES ($1, $2, $3)", userId, packUrl, vote.Upvote)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	} else {
		var upvote bool

		err = state.Pool.QueryRow(d.Context, "SELECT upvote FROM pack_votes WHERE user_id = $1 AND url = $2", userId, packUrl).Scan(&upvote)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		if upvote == vote.Upvote {
			var msg = "You have already upvoted this bot"

			if !vote.Upvote {
				msg = "You have already downvoted this bot"
			}

			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: msg,
					Error:   true,
				},
			}
		}

		// Update the vote
		_, err = state.Pool.Exec(d.Context, "UPDATE pack_votes SET upvote = $1, created_at = NOW() WHERE user_id = $2 AND url = $3", vote.Upvote, userId, packUrl)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.DefaultResponse(http.StatusNoContent)
}
