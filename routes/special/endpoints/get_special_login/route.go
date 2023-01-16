package get_special_login

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/routes/special/assets"
	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/crypto"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Special Login",
		Description: "This endpoint is used for special login actions. For example, data requests/deletions and regenerating tokens",
		Req:         assets.Action{},
		Resp:        assets.Redirect{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	if state.Config.HighSecurityCtx.Disabled {
		return api.HttpResponse{
			Status: http.StatusConflict,
			Json: types.ApiError{
				Error:   true,
				Message: "High security mode is disabled right now. Please try again later.",
			},
		}
	}

	// Read assets.Action to get the action
	var action assets.Action

	hresp, ok := api.MarshalReq(r, &action)

	if !ok {
		return hresp
	}

	if action.Action == "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Invalid action",
			},
		}
	}

	action.Nonce = ""
	action.Time = time.Now()

	if action.TID != "" {
		_, err := strconv.ParseInt(action.TID, 10, 64)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: "Invalid tid",
				},
			}
		}
	}

	var b bytes.Buffer
	e := json.NewEncoder(&b)

	err := e.Encode(action)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Store in redis
	stateTok := crypto.RandString(64)
	err = state.Redis.Set(d.Context, "spec:"+stateTok, b.Bytes(), 5*time.Minute).Err()

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: assets.Redirect{
			Redirect: "https://discord.com/api/oauth2/authorize?client_id=" + state.Config.HighSecurityCtx.ClientID + "&scope=identify&response_type=code&redirect_uri=" + state.Config.HighSecurityCtx.RedirectURL + "&state=" + stateTok,
		},
	}
}
