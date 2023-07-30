package post_bot_stats

import (
	"net/http"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Post Bot Stats",
		Description: "This endpoint posts the stats of a bot.",
		Req:         types.BotStats{},
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.BotStats

	resp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return resp
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	_, err = tx.Exec(d.Context, "UPDATE bots SET last_stats_post = NOW() WHERE bot_id = $1", d.Auth.ID)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if payload.Servers > 0 {
		_, err := tx.Exec(d.Context, "UPDATE bots SET servers = $1 WHERE bot_id = $2", payload.Servers, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Shards > 0 {
		_, err := tx.Exec(d.Context, "UPDATE bots SET shards = $1 WHERE bot_id = $2", payload.Shards, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Users > 0 {
		_, err := tx.Exec(d.Context, "UPDATE bots SET users = $1 WHERE bot_id = $2", payload.Users, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if len(payload.ShardList) > 0 {
		_, err := tx.Exec(d.Context, "UPDATE bots SET shard_list = $1 WHERE bot_id = $2", payload.ShardList, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Commit the transaction
	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	utils.ClearBotCache(d.Context, d.Auth.ID)

	return uapi.DefaultResponse(http.StatusNoContent)
}
