package create_blog_post

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-playground/validator/v10"
)

type CreateBlogPost struct {
	Slug        string   `db:"slug" json:"slug" validate:"required"`
	Title       string   `db:"title" json:"title" validate:"required"`
	Description string   `db:"description" json:"description" validate:"required"`
	Content     string   `db:"content" json:"content" validate:"required"`
	Tags        []string `db:"tags" json:"tags" validate:"required,dive,required"`
}

var compiledMessages = api.CompileValidationErrors(CreateBlogPost{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Blog Post",
		Description: "Creates a blog post. You must be an `iblhdev` or an `hadmin` to create a blog post. Returns a 204 on success",
		Req:         CreateBlogPost{},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user who is creating the blog post.",
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

	var payload CreateBlogPost

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

	if strings.Contains(payload.Slug, " ") {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Slug cannot contain spaces",
				Error:   true,
			},
		}
	}

	// Check slug
	var slugExists bool

	err = state.Pool.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM blogs WHERE slug = $1)", payload.Slug).Scan(&slugExists)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if slugExists {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Slug already exists",
				Error:   true,
			},
		}
	}

	// Create the blog post
	_, err = state.Pool.Exec(
		d.Context,
		"INSERT INTO blogs (slug, title, description, content, draft, user_id, tags) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		payload.Slug,
		payload.Title,
		payload.Description,
		payload.Content,
		true,
		d.Auth.ID,
		payload.Tags,
	)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
