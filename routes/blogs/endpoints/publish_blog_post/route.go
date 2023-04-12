package publish_blog_post

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Publish Blog Post",
		Description: "Publishes or unpublishes a blog post. You must be an `iblhdev` or an `hadmin` to create a blog post. Returns a 204 on success",
		Req:         types.PublishBlogPost{},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user who is creating the blog post.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "slug",
				Description: "The slug of the blog post.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Check if user is iblhdev or hadmin
	var iblhdev bool
	var hadmin bool

	err := state.Pool.QueryRow(d.Context, "SELECT iblhdev, hadmin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&iblhdev, &hadmin)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !iblhdev && !hadmin {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You do not have permission to publish a blog post",
				Error:   true,
			},
		}
	}

	var payload types.PublishBlogPost

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	slug := chi.URLParam(r, "slug")

	// Check slug
	var slugExists bool

	err = state.Pool.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM blogs WHERE slug = $1)", slug).Scan(&slugExists)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !slugExists {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "The slug does not exist",
				Error:   true,
			},
		}
	}

	// Publish the blog post
	_, err = state.Pool.Exec(d.Context, "UPDATE blogs SET draft = $1 WHERE slug = $2", payload.Draft, slug)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
