package patch_bot_vanity

import (
	"net/http"
	"strings"
	"unicode"

	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

type VanityUpdate struct {
	Vanity string `json:"vanity"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Bot Vanity",
		Description: "Updates a bots vanity. Returns 204 on success",
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
		Req:  VanityUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionSetBotVanity) {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to set bot vanity", Error: true},
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
			Json:   types.ApiError{Message: "Vanity cannot be empty", Error: true},
		}
	}

	if vanity.Vanity == "undefined" || vanity.Vanity == "null" || vanity.Vanity == "blog" || vanity.Vanity == "help" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot be undefined, blog, help or null", Error: true},
		}
	}

	vanity.Vanity = strings.TrimSuffix(vanity.Vanity, "-")

	vanity.Vanity = strings.ToLower(vanity.Vanity)

	vanity.Vanity = strings.ReplaceAll(vanity.Vanity, " ", "-")

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
			Json:   types.ApiError{Message: "Vanity is already taken", Error: true},
		}
	}

	// Update vanity
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET vanity = $1 WHERE bot_id = $2", vanity.Vanity, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	utils.ClearBotCache(d.Context, id)

	return api.DefaultResponse(http.StatusNoContent)
}
