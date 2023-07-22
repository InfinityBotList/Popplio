package redeem_payment_offer

import (
	"net/http"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/infinitybotlist/eureka/uapi/ratelimit"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(assets.PerkData{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Redeem Payment Offer",
		Description: "Redeems a payment offer for a user given a redeem code",
		Req:         assets.CreatePerkData{},
		Resp:        assets.RedirectUser{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "code",
				Description: "Redeem Code. Default codes: BOOSTPREMIUM -> special booster offer",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	code := r.URL.Query().Get("code")

	if code == "" {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Error: No code provided",
			},
			Status: http.StatusBadRequest,
		}
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var create assets.CreatePerkData

	hresp, ok := uapi.MarshalReqWithHeaders(r, &create, limit.Headers())

	if !ok {
		return hresp
	}

	payload := create.Parse(d.Auth.ID)

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	perk, err := assets.FindPerks(d.Context, payload)

	if err != nil {
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Error: " + err.Error(),
			},
		}
	}

	switch code {
	case "BOOSTPREMIUM":
		// Ensure bronze is the perk
		if perk.ID != "bronze" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "This offer code is only valid for the bronze plan",
				},
			}
		}

		// Check that the user is in fact a booster
		bs := utils.CheckUserBoosterStatus(d.Auth.ID)

		if !bs.IsBooster {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "This offer code is only valid for boosters",
				},
			}
		}

		err = assets.GivePerks(d.Context, payload)

		if err != nil {
			state.Logger.Error(err)
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "Error: " + err.Error(),
				},
			}
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusBadRequest,
		Json: types.ApiError{
			Message: "Invalid offer code",
		},
	}
}
