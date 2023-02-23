package patch_pack

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = api.CompileValidationErrors(PatchPack{})

type PatchPack struct {
	Name  string   `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	Short string   `json:"short" validate:"required,min=10,max=100" msg:"Description must be between 10 and 100 characters"`
	Tags  []string `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	Bots  []string `json:"bots" validate:"required,unique,min=1,max=10,dive,numeric" msg:"There must be between 1 and 10 bots without duplicates"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Pack",
		Description: "Edits a pack you are owner of based on the URL only. Returns 204 on success",
		Req:         PatchPack{},
		Resp:        types.AllPacks{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "id",
				Description: "The pack's URL",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload PatchPack

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	var id = chi.URLParam(r, "id")

	// Check that the pack exists
	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", id).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Check that the user is the owner of the pack
	var owner string

	err = state.Pool.QueryRow(d.Context, "SELECT owner FROM packs WHERE url = $1", id).Scan(&owner)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if owner != d.Auth.ID {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You are not the owner of this pack",
				Error:   true,
			},
		}
	}

	// Check that all bots exist
	for _, bot := range payload.Bots {
		botUser, err := utils.GetDiscordUser(d.Context, bot)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "One of the bot you wish to add does not exist [" + bot + "]: " + err.Error(),
					Error:   true,
				},
			}
		}

		if !botUser.Bot {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "One of the bot you wish to add is not actually a bot [" + bot + "]",
					Error:   true,
				},
			}
		}
	}

	// Update the pack
	_, err = state.Pool.Exec(d.Context, "UPDATE packs SET name = $1, short = $2, tags = $3, bots = $4 WHERE url = $5", payload.Name, payload.Short, payload.Tags, payload.Bots, id)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
