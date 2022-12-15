package post_stats

import (
	"net/http"
	"strconv"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/constants"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/state"
	"github.com/infinitybotlist/popplio/types"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:  "POST",
		Path:    "/bots/stats",
		OpId:    "post_stats",
		Summary: "Post Bot Stats",
		Description: `
This endpoint can be used to post the stats of a bot.

The variation` + constants.BackTick + `/bots/{bot_id}/stats` + constants.BackTick + ` can also be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**

**Example:**

` + constants.BackTick + constants.BackTick + constants.BackTick + `py
import requests

req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})

print(req.json())
` + constants.BackTick + constants.BackTick + constants.BackTick + "\n\n",
		Tags:     []string{api.CurrentTag},
		Req:      types.BotStatsDocs{},
		Resp:     types.ApiError{},
		AuthType: []types.TargetType{types.TargetTypeBot},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := d.Auth.ID

	var payload types.BotStats

	_, ok := api.MarshalReq(r, &payload)

	if !ok {
		if r.URL.Query().Get("count") != "" {
			payload = types.BotStats{}
		} else {
			return api.HttpResponse{
				Data:   constants.BadRequestStats,
				Status: http.StatusBadRequest,
			}
		}
	}

	if r.URL.Query().Get("count") != "" {
		count, err := strconv.ParseUint(r.URL.Query().Get("count"), 10, 32)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusBadRequest)
		}

		var countAny any = count

		payload.Count = &countAny
	}

	var err error

	servers, shards, users := payload.GetStats()

	if servers > 0 {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET servers = $1 WHERE bot_id = $2", servers, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if shards > 0 {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET shards = $1 WHERE bot_id = $2", shards, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if users > 0 {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET users = $1 WHERE bot_id = $2", users, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Get name and vanity, delete from cache
	var vanity string

	state.Pool.QueryRow(d.Context, "SELECT lower(vanity) FROM bots WHERE bot_id = $1", id).Scan(&vanity)

	// Delete from cache
	state.Redis.Del(d.Context, "bc-"+vanity)
	state.Redis.Del(d.Context, "bc-"+id)

	return api.HttpResponse{
		Data: constants.Success,
	}
}
