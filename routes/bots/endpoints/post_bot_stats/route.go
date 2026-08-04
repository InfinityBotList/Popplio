// Package post_bot_stats implements POST /bots/stats — "Post Bot Stats".
//
// This endpoint posts the stats of a bot. `status` is optional and
// self-reports the bot's presence (online/idle/dnd/offline) — post it
// periodically to keep it fresh, it will otherwise keep showing the last
// value posted.
package post_bot_stats

import (
	"net/http"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.BotStats{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Post Bot Stats",
		Description: "This endpoint posts the stats of a bot. `status` is optional and self-reports the bot's presence (online/idle/dnd/offline) — post it periodically to keep it fresh, it will otherwise keep showing the last value posted.",
		Req:         types.BotStats{},
		Resp:        types.ApiError{},
	}
}

// statUpdate is one optional column of the bots row that a stats post may carry.
//
// Every field of types.BotStats is optional, and a bot that omits one must keep
// whatever value it last reported rather than have it reset to zero. present
// records whether the payload actually carried the field, so omitted fields are
// skipped instead of written.
type statUpdate struct {
	// column is the bots column to write. It is interpolated into the UPDATE
	// statement, so it must only ever be a constant from statUpdates.
	column string
	// value is the new value, bound as a query parameter.
	value any
	// present is whether the payload carried this field at all.
	present bool
}

// statUpdates lists the columns a stats post can touch, in the order they are
// applied. Adding a stat means adding a line here rather than another
// near-identical block of update-and-check.
func statUpdates(payload types.BotStats) []statUpdate {
	return []statUpdate{
		{"servers", payload.Servers, payload.Servers > 0},
		{"shards", payload.Shards, payload.Shards > 0},
		{"users", payload.Users, payload.Users > 0},
		{"shard_list", payload.ShardList, len(payload.ShardList) > 0},

		// Self-reported presence. Supplements the JAPI-based metadata refresh,
		// which doesn't cover presence, and the gateway-cache-derived status
		// dovewing normally returns (only populated when the bot shares a
		// guild with our own Discord client, which most listed bots don't).
		{"self_status", payload.Status, payload.Status != ""},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.BotStats

	marshalResp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return marshalResp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("botID", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	// Always bumped, even when the payload carries no stats at all: the
	// timestamp is what marks the bot as still reporting.
	_, err = tx.Exec(d.Context, "UPDATE bots SET last_stats_post = NOW() WHERE bot_id = $1", d.Auth.ID)

	if err != nil {
		return resp.Err("Error while updating last_stats_post", err, zap.String("botID", d.Auth.ID))
	}

	for _, update := range statUpdates(payload) {
		if !update.present {
			continue
		}

		_, err := tx.Exec(d.Context, "UPDATE bots SET "+update.column+" = $1 WHERE bot_id = $2", update.value, d.Auth.ID)

		if err != nil {
			return resp.Err("Error while updating "+update.column, err, zap.String("botID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("botID", d.Auth.ID), zap.Any("payload", payload))
	}

	return resp.NoContent()
}
