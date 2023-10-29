package get_bot_meta

import (
	"net/http"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/infinitybotlist/eureka/uapi/ratelimit"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot Metadata",
		Description: "Gets the metadata of a bot such as whether it is already in the database/bot id checks",
		Resp:        types.DiscordBotMeta{},
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 3,
		Bucket:      "get_bot_meta",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error("Error while ratelimiting", zap.Error(err), zap.String("bucket", "get_bot_meta"))
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

	fallbackId := r.URL.Query().Get("fallback_bot_id")
	cid := chi.URLParam(r, "cid")

	// Get bot metadata
	meta, err := assets.CheckBot(d.Context, fallbackId, cid)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	if meta == nil {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Internal error: meta returned nil"},
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   meta,
	}
}
