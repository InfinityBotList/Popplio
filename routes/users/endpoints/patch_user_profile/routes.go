// Package patch_user_profile implements PATCH /users/{id} — "Update User
// Profile".
//
// Updates a users profile. Returns 204 on success
package patch_user_profile

import (
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

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
		Req:  types.ProfileUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	// Fetch profile update from body
	var profile types.ProfileUpdate

	hresp, ok := uapi.MarshalReq(r, &profile)

	if !ok {
		return hresp
	}

	err := validators.ValidateExtraLinks(profile.ExtraLinks)

	if err != nil {
		return resp.BadRequest("Failed to validate extra links: " + err.Error())
	}

	if len(profile.About) > 1000 {
		return resp.BadRequest("About me is over 1000 characters!")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID))
	}

	_, err = tx.Exec(d.Context, "UPDATE users SET updated_at = NOW() WHERE user_id = $1", id)

	if err != nil {
		return resp.Err("Error while updating updated_at", err, zap.String("userID", d.Auth.ID))
	}

	// Update extra links
	_, err = tx.Exec(d.Context, "UPDATE users SET extra_links = $1 WHERE user_id = $2", profile.ExtraLinks, id)

	if err != nil {
		return resp.Err("Error while updating extra links", err, zap.String("userID", d.Auth.ID))
	}

	if profile.About != "" {
		_, err = tx.Exec(d.Context, "UPDATE users SET about = $1 WHERE user_id = $2", profile.About, id)

		if err != nil {
			return resp.Err("Error while updating about", err, zap.String("userID", d.Auth.ID))
		}
	}

	if profile.CaptchaSponsorEnabled != nil {
		_, err = tx.Exec(d.Context, "UPDATE users SET captcha_sponsor_enabled = $1 WHERE user_id = $2", *profile.CaptchaSponsorEnabled, id)

		if err != nil {
			return resp.Err("Error while updating captcha sponsor enabled", err, zap.String("userID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
