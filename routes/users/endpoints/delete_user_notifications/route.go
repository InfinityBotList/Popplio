package delete_user_notifications

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs(tagName string) {
	docs.Route(&docs.Doc{
		Method:      "DELETE",
		Path:        "/users/{id}/notification",
		OpId:        "delete_user_notifications",
		Summary:     "Delete User Notifications",
		Description: "Deletes a users notification",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "notif_id",
				Description: "Notification ID",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp:     types.ApiError{},
		Tags:     []string{tagName},
		AuthType: []string{"User"},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var id = chi.URLParam(r, "id")

	if id == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	// Check for notif_id
	if r.URL.Query().Get("notif_id") == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	// Fetch auth from postgresdb
	if r.Header.Get("Authorization") == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
		return
	} else {
		authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

		if authId == nil || *authId != id {
			d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
			return
		}
	}

	_, err := state.Pool.Exec(d.Context, "DELETE FROM poppypaw WHERE user_id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id"))

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	d.Resp <- types.HttpResponse{
		Status: http.StatusNoContent,
	}

}
