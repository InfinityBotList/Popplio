package patch_user_profile

import (
	"encoding/json"
	"io"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PATCH",
		Path:        "/users/{id}",
		OpId:        "patch_user_profile",
		Summary:     "Update User Profile",
		Description: "Updates a users profile",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.ProfileUpdate{},
		Resp: types.ApiError{},
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Fetch profile update from body
	var profile types.ProfileUpdate

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bodyBytes, &profile)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	err = utils.ValidateExtraLinks(profile.ExtraLinks)

	if err != nil {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Hmmm... " + err.Error(),
			},
		}
		return
	}

	// Update extra links
	_, err = state.Pool.Exec(d.Context, "UPDATE users SET extra_links = $1 WHERE user_id = $2", profile.ExtraLinks, id)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	if profile.About != "" {
		if len(profile.About) > 1000 {
			d.Resp <- types.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   `{"error":true,"message":"About me is over 1000 characters!"}`,
			}
			return
		}

		// Update about
		_, err = state.Pool.Exec(d.Context, "UPDATE users SET about = $1 WHERE user_id = $2", profile.About, id)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
			return
		}
	}

	state.Redis.Del(d.Context, "uc-"+id)

	d.Resp <- types.HttpResponse{
		Status: http.StatusNoContent,
	}
}
