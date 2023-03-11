package post_stats

import (
	"net/http"
	"reflect"
	"strconv"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
)

func GetStats(s types.BotStats) (servers uint64, shards uint64, users uint64) {
	var serverCount any
	var shardCount any
	var userCount any

	if s.Servers != nil {
		serverCount = *s.Servers
	} else if s.GuildCount != nil {
		serverCount = *s.GuildCount
	} else if s.ServerCount != nil {
		serverCount = *s.ServerCount
	} else if s.Count != nil {
		serverCount = *s.Count
	} else if s.Guilds != nil {
		serverCount = *s.Guilds
	}

	if s.Shards != nil {
		shardCount = *s.Shards
	} else if s.ShardCount != nil {
		shardCount = *s.ShardCount
	}

	if s.Users != nil {
		userCount = *s.Users
	} else if s.UserCount != nil {
		userCount = *s.UserCount
	}

	var serversParsed uint64
	var shardsParsed uint64
	var usersParsed uint64

	// Handle uint64 by converting to uint32
	if serverInt, ok := serverCount.(uint64); ok {
		serversParsed = serverInt
	}

	if shardInt, ok := shardCount.(uint64); ok {
		shardsParsed = shardInt
	}
	if userInt, ok := userCount.(uint64); ok {
		usersParsed = userInt
	}

	// Handle uint32
	if serverInt, ok := serverCount.(uint32); ok {
		serversParsed = uint64(serverInt)
	}
	if shardInt, ok := shardCount.(uint32); ok {
		shardsParsed = uint64(shardInt)
	}
	if userInt, ok := userCount.(uint32); ok {
		usersParsed = uint64(userInt)
	}

	// Handle float64
	if serverFloat, ok := serverCount.(float64); ok {
		if serverFloat < 0 {
			serversParsed = 0
		} else {
			serversParsed = uint64(serverFloat)
		}
	}
	if shardFloat, ok := shardCount.(float64); ok {
		if shardFloat < 0 {
			shardsParsed = 0
		} else {
			shardsParsed = uint64(shardFloat)
		}
	}
	if userFloat, ok := userCount.(float64); ok {
		if userFloat < 0 {
			userFloat = 0
		} else {
			usersParsed = uint64(userFloat)
		}
	}

	// Handle float32
	if serverFloat, ok := serverCount.(float32); ok {
		serversParsed = uint64(serverFloat)
	}
	if shardFloat, ok := shardCount.(float32); ok {
		shardsParsed = uint64(shardFloat)
	}
	if userFloat, ok := userCount.(float32); ok {
		usersParsed = uint64(userFloat)
	}

	// Handle int64
	if serverInt, ok := serverCount.(int64); ok {
		if serverInt < 0 {
			serversParsed = 0
		} else {
			serversParsed = uint64(serverInt)
		}
	}
	if shardInt, ok := shardCount.(int64); ok {
		if shardInt < 0 {
			shardsParsed = 0
		} else {
			shardsParsed = uint64(shardInt)
		}
	}
	if userInt, ok := userCount.(int64); ok {
		if userInt < 0 {
			usersParsed = 0
		} else {
			usersParsed = uint64(userInt)
		}
	}

	// Handle int32
	if serverInt, ok := serverCount.(int32); ok {
		if serverInt < 0 {
			serversParsed = 0
		} else {
			serversParsed = uint64(serverInt)
		}
	}
	if shardInt, ok := shardCount.(int32); ok {
		if shardInt < 0 {
			shardsParsed = 0
		} else {
			shardsParsed = uint64(shardInt)
		}
	}
	if userInt, ok := userCount.(int32); ok {
		if userInt < 0 {
			usersParsed = 0
		} else {
			usersParsed = uint64(userInt)
		}
	}

	// Handle string
	if serverString, ok := serverCount.(string); ok {
		if serverString == "" {
			serversParsed = 0
		} else {
			serversParsed, _ = strconv.ParseUint(serverString, 10, 64)
		}
	}

	if shardString, ok := shardCount.(string); ok {
		if shardString == "" {
			shardsParsed = 0
		} else {
			shardsParsed, _ = strconv.ParseUint(shardString, 10, 64)
		}
	}

	if userString, ok := userCount.(string); ok {
		if userString == "" {
			usersParsed = 0
		} else {
			usersParsed, _ = strconv.ParseUint(userString, 10, 64)
		}
	}

	state.Logger.With(
		"serverCount", serversParsed,
		"shardCount", shardsParsed,
		"userCount", usersParsed,
		"serversType", reflect.TypeOf(serverCount),
		"shardsType", reflect.TypeOf(shardCount),
		"usersType", reflect.TypeOf(userCount),
	).Info("Parsed stats")

	return serversParsed, shardsParsed, usersParsed
}

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
		Req:  types.BotStatsDocs{},
		Resp: types.ApiError{},
	}
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
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: "Slow down, bucko! You're not posting stats correctly. Try posting stats as integers and not as strings?",
				},
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

	var rowcount int64

	var err error

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", id).Scan(&rowcount)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if rowcount == 0 || rowcount > 1 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	servers, shards, users := GetStats(payload)

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

	utils.ClearBotCache(d.Context, id)

	return api.DefaultResponse(http.StatusNoContent)
}
