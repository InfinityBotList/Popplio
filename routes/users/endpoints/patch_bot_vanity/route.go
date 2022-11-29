package patch_bot_vanity

import (
	"io"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type VanityUpdate struct {
	Vanity string `json:"vanity"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PATCH",
		Path:        "/users/{uid}/bots/{bid}/vanity",
		OpId:        "patch_user_profile",
		Summary:     "Update User Profile",
		Description: "Updates a users profile. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
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

func Route(d api.RouteData, r *http.Request) {
	botId := chi.URLParam(r, "bid")
	// Read vanity from body
	var vanity VanityUpdate

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bodyBytes, &vanity)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	// Strip out unicode characters
	vanity.Vanity = strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, vanity.Vanity)

	if vanity.Vanity == "" {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot be empty"},
		}
		return
	}

	vanity.Vanity = strings.TrimSuffix(vanity.Vanity, "-")

	vanity.Vanity = strings.ToLower(vanity.Vanity)

	// Ensure vanity doesn't already exist
	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE lower(vanity) = $1", vanity.Vanity).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if count > 0 {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity is already taken"},
		}
		return
	}

	// Update vanity
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET vanity = $1 WHERE bot_id = $2", vanity.Vanity, botId)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	d.Resp <- api.DefaultResponse(http.StatusNoContent)
}
