package patch_bot_vanity

import (
	"net/http"
	"strings"
	"unicode"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

type VanityUpdate struct {
	Vanity string `json:"vanity"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PATCH",
		Path:        "/users/{uid}/bots/{bid}/vanity",
		OpId:        "patch_bot_vanity",
		Summary:     "Update User Vanity",
		Description: "Updates a users vanity. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:      VanityUpdate{},
		Resp:     types.ApiError{},
		Tags:     []string{api.CurrentTag},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	botId := chi.URLParam(r, "bid")

	// Validate that they actually own this bot
	isOwner, err := utils.IsBotOwner(d.Context, d.Auth.ID, botId)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	if !isOwner {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You do not own the bot you are trying to manage"},
		}
	}

	// Read vanity from body
	var vanity VanityUpdate

	hresp, ok := api.MarshalReq(r, &vanity)

	if !ok {
		return hresp
	}

	// Strip out unicode characters
	vanity.Vanity = strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, vanity.Vanity)

	if vanity.Vanity == "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot be empty"},
		}
	}

	vanity.Vanity = strings.TrimSuffix(vanity.Vanity, "-")

	vanity.Vanity = strings.ToLower(vanity.Vanity)

	// Ensure vanity doesn't already exist
	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE lower(vanity) = $1", vanity.Vanity).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count > 0 {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity is already taken"},
		}
	}

	// Update vanity
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET vanity = $1 WHERE bot_id = $2", vanity.Vanity, botId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
