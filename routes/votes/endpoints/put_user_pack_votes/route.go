package put_user_pack_votes

import (
	"net/http"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

type CreatePackVote struct {
	Upvote bool `json:"upvote"`
	Clear  bool `json:"clear,omitempty"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:  "PUT",
		Path:    "/users/{uid}/packs/{url}/votes",
		OpId:    "put_user_pack_votes",
		Summary: "Create User Pack Vote",
		Description: `Creates a vote for a pack. 
		
This updates any existing vote or creates a new one if none exist. 

Does NOT error if the same vote is sent twice but will merely have no effect. Use` + constants.DoubleBackTick + `clear` + constants.DoubleBackTick + ` to clear a vote (which overrides upvote if sent). 

Returns 204 on success.`,
		Tags: []string{api.CurrentTag},
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
		AuthType: []types.TargetType{types.TargetTypeUser},
		Req:      CreatePackVote{},
		Resp:     types.ApiError{},
	})
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
			Data:   constants.VoteBanned,
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
		// Update the vote
		_, err = state.Pool.Exec(d.Context, "UPDATE pack_votes SET upvote = $1, created_at = NOW() WHERE user_id = $2 AND url = $3", vote.Upvote, userId, packUrl)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
