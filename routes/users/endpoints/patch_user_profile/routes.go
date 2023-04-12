package patch_user_profile

import (
	"io"
	"net/http"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type ProfileUpdate struct {
	About                 string       `json:"bio"`
	ExtraLinks            []types.Link `json:"extra_links"`
	CaptchaSponsorEnabled *bool        `json:"captcha_sponsor_enabled"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
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
		Req:  ProfileUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := chi.URLParam(r, "id")

	// Fetch profile update from body
	var profile ProfileUpdate

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = json.Unmarshal(bodyBytes, &profile)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = utils.ValidateExtraLinks(profile.ExtraLinks)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Hmmm... " + err.Error(),
			},
		}
	}

	// Update extra links
	_, err = state.Pool.Exec(d.Context, "UPDATE users SET extra_links = $1 WHERE user_id = $2", profile.ExtraLinks, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if profile.About != "" {
		if len(profile.About) > 1000 {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   `{"error":true,"message":"About me is over 1000 characters!"}`,
			}
		}

		// Update about, captcha_sponsor_enabled
		_, err = state.Pool.Exec(d.Context, "UPDATE users SET about = $1 WHERE user_id = $2", profile.About, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if profile.CaptchaSponsorEnabled != nil {
		_, err = state.Pool.Exec(d.Context, "UPDATE users SET captcha_sponsor_enabled = $1 WHERE user_id = $2", *profile.CaptchaSponsorEnabled, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	state.Redis.Del(d.Context, "uc-"+id)

	return api.DefaultResponse(http.StatusNoContent)
}
