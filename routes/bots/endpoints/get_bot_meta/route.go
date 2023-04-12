package get_bot_meta

import (
	"net/http"
	"popplio/api"
	"popplio/ratelimit"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot Metadata",
		Description: "Gets the metadata of a bot such as whether it is already in the database/bot id checks",
		Req:         assets.DiscordBotMeta{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID for authentication",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "cid",
				Description: "The client ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "fallback_bot_id",
				Description: "The fallback bot ID to use if japi.rest is offline",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 3,
		Bucket:      "get_bot_meta",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	fallbackId := r.URL.Query().Get("fallback_bot_id")
	cid := chi.URLParam(r, "cid")

	// Get bot metadata
	meta, err := assets.CheckBot(fallbackId, cid)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: err.Error(),
				Error:   true,
			},
		}
	}

	if meta == nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal error: meta returned nil",
				Error:   true,
			},
		}
	}

	return api.HttpResponse{
		Status: http.StatusOK,
		Json:   meta,
	}
}
