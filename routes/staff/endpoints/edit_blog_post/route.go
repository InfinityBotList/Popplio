package edit_blog_post

import (
	"net/http"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.EditBlogPost{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Blog Post",
		Description: "Edits a blog post. You must be an `iblhdev` or an `hadmin` to edit a blog post. Returns a 204 on success",
		Req:         types.EditBlogPost{},
		Params: []docs.Parameter{
			{
				Name:        "slug",
				Description: "The slug of the blog post to edit.",
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
			Json:   types.ApiError{Message: "You do not have permission to create a blog post"},
		}
	}

	var payload types.EditBlogPost

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	slug := chi.URLParam(r, "slug")

	// Check slug
	var slugExists bool

	err = state.Pool.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM blogs WHERE slug = $1)", slug).Scan(&slugExists)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !slugExists {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Slug does not exist"},
		}
	}

	// Update the blog post
	if payload.Title != "" {
		_, err = state.Pool.Exec(
			d.Context,
			"UPDATE blogs SET title = $1 WHERE slug = $2",
			payload.Title,
			slug,
		)
	}
	if payload.Description != "" {
		_, err = state.Pool.Exec(
			d.Context,
			"UPDATE blogs SET description = $1 WHERE slug = $2",
			payload.Description,
			slug,
		)
	}

	if payload.Content != "" {
		_, err = state.Pool.Exec(
			d.Context,
			"UPDATE blogs SET content = $1 WHERE slug = $2",
			payload.Content,
			slug,
		)
	}

	if payload.Tags != nil {
		_, err = state.Pool.Exec(
			d.Context,
			"UPDATE blogs SET tags = $1 WHERE slug = $2",
			payload.Tags,
			slug,
		)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
