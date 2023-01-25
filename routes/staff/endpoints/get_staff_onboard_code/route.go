package get_staff_onboard_code

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Staff Onboard Code",
		Description: "Gets a staff onboard code",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "frag",
				Description: "Onboard Code Fragment",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.StaffOnboardCode{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	if name == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	frag := r.URL.Query().Get("frag")

	if frag == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users WHERE user_id = $1", name).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var code string

	err = state.Pool.QueryRow(d.Context, "SELECT staff_onboard_session_code FROM users WHERE user_id = $1", name).Scan(&code)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Ensure first 20 chars of code is equal to frag
	if len(code) < 20 || code[:20] != frag {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "invalid fragment, check link?",
				Error:   true,
			},
		}
	}

	// Split code by @
	codeTimeArr := strings.Split(code, "@")

	if len(codeTimeArr) != 2 {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "database contains invalid code (format invalid)",
				Error:   true,
			},
		}
	}

	codeTime := codeTimeArr[1]

	// Check if code is expired
	codeTimeInt, err := strconv.ParseInt(codeTime, 10, 64)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "database contains invalid code (time invalid)",
				Error:   true,
			},
		}
	}

	if time.Now().Unix()-codeTimeInt > 3600 {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "staff onboard code expired",
				Error:   true,
			},
		}
	}

	return api.HttpResponse{
		Json: types.StaffOnboardCode{
			Code: codeTimeArr[0],
		},
	}
}
