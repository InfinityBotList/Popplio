package delete_pack

import (
	"net/http"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Pack",
		Description: "Deletes a pack by URL. You *must* be the owner of the pack to delete packs. Returns 204 on success",
		Resp:        types.ApiError{},
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	// Check that the pack exists
	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", id).Scan(&count)

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Check that the user is the owner of the pack
	var owner string

	err = state.Pool.QueryRow(d.Context, "SELECT owner FROM packs WHERE url = $1", id).Scan(&owner)

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if owner != d.Auth.ID {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are not the owner of this pack"},
		}
	}

	// Delete the pack
	_, err = state.Pool.Exec(d.Context, "DELETE FROM packs WHERE url = $1", id)

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
