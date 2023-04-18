package post_stats

import (
	"net/http"

	"popplio/constants"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Post Bot Stats",
		Description: `
This endpoint can be used to post the stats of a bot. This endpoint does not resolve the ID.

**Example:**

` + constants.BackTick + constants.BackTick + constants.BackTick + `py
import requests

req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})

print(req.json())
` + constants.BackTick + constants.BackTick + constants.BackTick + "\n\n",
		Req:  types.BotStats{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.BotStats

	resp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return resp
	}

	if payload.Servers > 0 {
		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET servers = $1 WHERE bot_id = $2", payload.Servers, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Shards > 0 {
		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET shards = $1 WHERE bot_id = $2", payload.Shards, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Users > 0 {
		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET users = $1 WHERE bot_id = $2", payload.Users, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if len(payload.ShardList) > 0 {
		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET shard_list = $1 WHERE bot_id = $2", payload.ShardList, d.Auth.ID)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	utils.ClearBotCache(d.Context, d.Auth.ID)

	return uapi.DefaultResponse(http.StatusNoContent)
}
