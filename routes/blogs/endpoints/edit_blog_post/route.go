package edit_blog_post

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type EditBlogPost struct {
	Title       string   `db:"title" json:"title" validate:"required"`
	Description string   `db:"description" json:"description" validate:"required"`
	Content     string   `db:"content" json:"content" validate:"required"`
	Tags        []string `db:"tags" json:"tags" validate:"required,dive,required"`
}

var compiledMessages = api.CompileValidationErrors(EditBlogPost{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edits Blog Post",
		Description: "Edits a blog post. You must be an `iblhdev` or an `hadmin` to edit a blog post. Returns a 204 on success",
		Req:         EditBlogPost{},
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
				Description: "The slug of the blog post to edit.",
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
				Message: "You do not have permission to create a blog post",
				Error:   true,
			},
		}
	}

	var payload EditBlogPost

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
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
				Message: "Slug does not exist",
				Error:   true,
			},
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
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
