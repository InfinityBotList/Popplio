package assets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/types"
	"strconv"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"go.uber.org/zap"
)

type japidata struct {
	Cached bool `json:"cached"`
	Data   struct {
		Message     string `json:"message,omitempty"`
		Application *struct {
			ID          string   `json:"id"`
			BotPublic   bool     `json:"bot_public"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		} `json:"application"`
		Bot *struct {
			ID                    string   `json:"id"`
			ApproximateGuildCount int      `json:"approximate_guild_count"`
			Username              string   `json:"username"`
			AvatarURL             string   `json:"avatarURL"`
			AvatarHash            string   `json:"avatarHash"`
			PublicFlagsArray      []string `json:"public_flags_array"`
		} `json:"bot"`
	} `json:"data"`
}

func CheckBot(ctx context.Context, fallbackBotId, clientId string) (*types.DiscordBotMeta, error) {
	var fetchErrors = map[string]string{}

	// Convert client id to int
	cidInt, err := strconv.ParseInt(clientId, 10, 64)

	if err != nil {
		return nil, fmt.Errorf("error parsing client id as int: %s", clientId)
	}

	cli := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://japi.rest/discord/v1/application/"+clientId, nil)

	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err.Error())
	}

	japiKey := state.Config.JAPI.Key
	if japiKey != "" {
		req.Header.Set("Authorization", japiKey)
	}

	resp, err := cli.Do(req)

	if err != nil {
		fetchErrors["japi.doError"] = err.Error()
		resp = &http.Response{
			Status:     "418 I'm a teapot",
			StatusCode: http.StatusTeapot,
		}
	}

	var metadata *types.DiscordBotMeta
	switch {
	// 429, fallback
	case resp.StatusCode == http.StatusTooManyRequests:
		fetchErrors["japi.ratelimit"] = fmt.Sprintf("we're being ratelimited by our anti-abuse provider! Please try again in %s seconds", resp.Header.Get("Retry-After"))
	// 5** server error fallback
	case resp.StatusCode > 500:
		fetchErrors["japi.server"] = fmt.Sprintf("the JAPI server is having issues! Status code: %d", resp.StatusCode)
	// 408, fallback
	case resp.StatusCode == http.StatusRequestTimeout:
		fetchErrors["japi.timeout"] = "the JAPI server is taking too long to respond!"
	// 418, fallback
	case resp.StatusCode == http.StatusTeapot:
		fetchErrors["japi.status"] = "the JAPI server did not respond to the request correctly!"
	// 4** client error should not be fallback'd
	case resp.StatusCode > 400:
		return nil, fmt.Errorf("we couldn't find a bot with that client ID! Status code: %d", resp.StatusCode)
	// 2** is an invalid response, fallback
	case resp.StatusCode > 200:
		fetchErrors["japi.status"] = fmt.Sprintf("the JAPI server returned an invalid status code! Status code: %d", resp.StatusCode)
	case resp.StatusCode == 200:
		defer resp.Body.Close()

		var data japidata

		err = json.NewDecoder(resp.Body).Decode(&data)

		if err != nil {
			return nil, err
		}

		if data.Data.Message != "" {
			fetchErrors["japi.message"] = data.Data.Message
		}

		if data.Data.Bot == nil || data.Data.Application == nil {
			return nil, errors.New("woah there, we found an application with no associated bot?")
		}

		if data.Data.Bot.ID == "" {
			return nil, errors.New("woah there, we found an application with no associated bot?")
		}

		if !data.Cached {
			state.Logger.With(
				zap.String("bot_id", data.Data.Bot.ID),
				zap.String("client_id", clientId),
			).Info("JAPI cache MISS")
		} else {
			state.Logger.With(
				zap.String("bot_id", data.Data.Bot.ID),
				zap.String("client_id", clientId),
			).Info("JAPI cache HIT")
		}

		user, err := dovewing.GetUser(ctx, data.Data.Bot.ID, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, errors.New("please contact support, an error has occured while trying to fetch basic info")
		}

		metadata = &types.DiscordBotMeta{
			BotID:       data.Data.Bot.ID,
			ClientID:    clientId,
			Name:        user.Username,
			GuildCount:  data.Data.Bot.ApproximateGuildCount,
			BotPublic:   data.Data.Application.BotPublic,
			Avatar:      user.Avatar,
			Flags:       data.Data.Bot.PublicFlagsArray,
			Description: data.Data.Application.Description,
			Tags:        data.Data.Application.Tags,
			FetchErrors: fetchErrors,
			Fallback:    false,
		}
	}

	if metadata == nil {
		// Fallback to RPC, but this is less accurate
		req, err := http.NewRequestWithContext(ctx, "GET", state.Config.Meta.PopplioProxy+"/api/v10/applications/"+clientId+"/rpc", nil)

		if err != nil {
			return nil, err
		}

		resp, err := cli.Do(req)

		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("we're being ratelimited by discord! Please try again in %s seconds", resp.Header.Get("Retry-After"))
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("we couldn't find a bot with that client ID! Status code: %d", resp.StatusCode)
		}

		defer resp.Body.Close()

		var rpcFallbackData struct {
			BotPublic bool `json:"bot_public"`
		}

		err = json.NewDecoder(resp.Body).Decode(&rpcFallbackData)

		if err != nil {
			return nil, err
		}

		if fallbackBotId == "" {
			if cidInt < 132550911590400000 {
				return nil, errors.New("fallbackNeeded")
			}

			fallbackBotId = clientId
		}

		// Check that the client id is also a bot (or rather, hope)
		user, err := dovewing.GetUser(ctx, fallbackBotId, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, errors.New("the client id provided is not an actual bot id")
		}

		metadata = &types.DiscordBotMeta{
			BotID:       fallbackBotId,
			ClientID:    clientId,
			Name:        user.Username,
			Avatar:      user.Avatar,
			BotPublic:   rpcFallbackData.BotPublic,
			FetchErrors: fetchErrors,
			Fallback:    true,
		}
	}

	// Check if the bot is already in the database
	var count int

	err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", metadata.BotID).Scan(&count)

	if err != nil {
		return nil, errors.New("failed to check if bot is already in the database")
	}

	if count > 0 {
		// Get bot type
		var listType string

		err = state.Pool.QueryRow(ctx, "SELECT type FROM bots WHERE bot_id = $1", metadata.BotID).Scan(&listType)

		if err != nil {
			return nil, errors.New("failed to get bot type")
		}

		if listType == "" {
			return nil, errors.New("list type is invalid, contact support")
		}

		metadata.ListType = listType
	}

	return metadata, nil
}
