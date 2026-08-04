// Package get_bot_meta implements GET /bots/{client_id}/meta — "Get Bot
// Metadata".
//
// Gets the metadata of a bot such as whether it is already in the
// database/bot id checks
package get_bot_meta

import (
	"net/http"
	"popplio/api/resp"
	"popplio/routes/bots/assets"
	"popplio/types"
	"time"

	"github.com/infinitybotlist/eureka/ratelimit"
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
				Name:        "client_id",
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
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "get_bot_meta"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	fallbackId := r.URL.Query().Get("fallback_bot_id")
	cid := chi.URLParam(r, "client_id")

	// Get bot metadata
	meta, err := assets.CheckBot(d.Context, fallbackId, cid)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	if meta == nil {
		return resp.ErrBody("Internal error: meta returned nil", "Internal error: meta returned nil", nil)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   meta,
	}
}
