// Package redeem_payment_offer implements POST
// /users/{id}/redeem-payment-offer — "Redeem Payment Offer".
//
// Redeems a payment offer for a user given a redeem code
package redeem_payment_offer

import (
	"net/http"
	"popplio/api/resp"
	"popplio/routes/payments/assets"
	"popplio/state"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/infinitybotlist/eureka/ratelimit"
	"go.uber.org/zap"

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
		return resp.BadRequest("Error: No code provided")
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "payments"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
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
		state.Logger.Error("Error while finding perk", zap.Error(err), zap.String("userID", d.Auth.ID))
		return resp.BadRequest("Error: " + err.Error())
	}

	switch code {
	case "BOOSTPREMIUM":
		// Ensure bronze is the perk
		if perk.ID != "bronze" {
			return resp.BadRequest("This offer code is only valid for the bronze plan")
		}

		// Check that the user is in fact a booster
		userId, err := snowflake.Parse(d.Auth.ID)

		if err != nil {
			state.Logger.Error("Error while parsing snowflake", zap.Error(err), zap.String("userID", d.Auth.ID))
			return resp.BadRequest("Error: " + err.Error())
		}

		bs := assets.CheckUserBoosterStatus(userId)

		if !bs.IsBooster {
			return resp.BadRequest("This offer code is only valid for boosters")
		}

		var lastRedeemedBoosterOffer *time.Time
		err = state.Pool.QueryRow(d.Context, "SELECT last_booster_claim FROM users WHERE user_id = $1", d.Auth.ID).Scan(&lastRedeemedBoosterOffer)

		if err != nil {
			state.Logger.Error("Error while checking last booster claim", zap.Error(err), zap.String("userID", d.Auth.ID))
			return resp.BadRequest("Error: " + err.Error())
		}

		// Check the last time the user redeemed a booster offer
		if lastRedeemedBoosterOffer != nil {
			if time.Since(*lastRedeemedBoosterOffer) < 30*24*time.Hour {
				return resp.BadRequest("You can only redeem a booster offer once every 30 days")
			}
		}

		err = assets.GivePerks(d.Context, payload)

		if err != nil {
			state.Logger.Error("Error while giving perks", zap.Error(err), zap.String("userID", d.Auth.ID))
			return resp.BadRequest("Error: " + err.Error())
		}
	}

	return resp.BadRequest("Invalid offer code")
}
