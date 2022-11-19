package post_stats

import (
	"encoding/json"
	"io"
	"net/http"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
)

func Docs() {
	docs.Route(&docs.Doc{
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
		AuthType: []string{"Bot"},
	})
}

func Route(d api.RouteData, r *http.Request) {
	if r.Body == nil {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	var id *string

	// Check token
	if r.Header.Get("Authorization") == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
		return
	} else {
		id = utils.AuthCheck(r.Header.Get("Authorization"), true)

		if id == nil {
			d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
			return
		}
	}

	defer r.Body.Close()

	var payload types.BotStats

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bodyBytes, &payload)

	if err != nil {
		if r.URL.Query().Get("count") != "" {
			payload = types.BotStats{}
		} else {
			state.Logger.Error(err)
			d.Resp <- types.HttpResponse{
				Data:   constants.BadRequestStats,
				Status: http.StatusBadRequest,
			}
			return
		}
	}

	if r.URL.Query().Get("count") != "" {
		count, err := strconv.ParseUint(r.URL.Query().Get("count"), 10, 32)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
			return
		}

		var countAny any = count

		payload.Count = &countAny
	}

	servers, shards, users := payload.GetStats()

	if servers > 0 {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET servers = $1 WHERE bot_id = $2", servers, id)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
			return
		}
	}

	if shards > 0 {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET shards = $1 WHERE bot_id = $2", shards, id)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
			return
		}
	}

	if users > 0 {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET users = $1 WHERE bot_id = $2", users, id)

		if err != nil {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
			return
		}
	}

	// Get name and vanity, delete from cache
	var vanity string

	state.Pool.QueryRow(d.Context, "SELECT lower(vanity) FROM bots WHERE bot_id = $1", id).Scan(&vanity)

	// Delete from cache
	state.Redis.Del(d.Context, "bc-"+vanity)
	state.Redis.Del(d.Context, "bc-"+*id)

	d.Resp <- types.HttpResponse{
		Data: constants.Success,
	}
}
