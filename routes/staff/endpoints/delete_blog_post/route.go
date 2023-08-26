package delete_blog_post

import (
	"net/http"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Blog List",
		Description: "Deletes a blog post on the list. You must be an `iblhdev` or an `hadmin` to delete a blog post.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user who is deleting the blog post.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "slug",
				Description: "The slug of the blog post to delete.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	// Check if user is iblhdev or hadmin
	var iblhdev bool
	var hadmin bool

	err = state.Pool.QueryRow(d.Context, "SELECT iblhdev, hadmin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&iblhdev, &hadmin)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !iblhdev && !hadmin {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to delete a blog post"},
		}
	}

	// Delete the blog post
	_, err = state.Pool.Exec(d.Context, "DELETE FROM blogs WHERE slug = $1", chi.URLParam(r, "slug"))

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
