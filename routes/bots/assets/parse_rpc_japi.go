package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"popplio/state"
	"strconv"
	"time"

	"github.com/infinitybotlist/dovewing"
	"go.uber.org/zap"
)

type DiscordBotMeta struct {
	BotID       string   `json:"bot_id" description:"The bot's ID"`
	ClientID    string   `json:"client_id" description:"The bot's client ID"`
	Name        string   `json:"name" description:"The bot's name"`
	Avatar      string   `json:"avatar" description:"The bot's avatar"`
	ListType    string   `json:"list_type" description:"If this is empty, then it is not on the list"`
	GuildCount  int      `json:"guild_count" description:"The bot's guild count"`
	BotPublic   bool     `json:"bot_public" description:"Whether or not the bot is public"`
	Flags       []string `json:"flags" description:"The bot's flags"`
	Description string   `json:"description" description:"The suggested description for the bot"`
	Tags        []string `json:"tags" description:"The suggested tags for the bot"`
	Fallback    bool     `json:"fallback" description:"Whether or not we had to fallback to RPC from JAPI.rest"`
}

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

func CheckBot(fallbackBotId, clientId string) (*DiscordBotMeta, error) {
	// Convert client id to int
	cidInt, err := strconv.ParseInt(clientId, 10, 64)

	if err != nil {
		return nil, fmt.Errorf("error parsing client id as int: %s", clientId)
	}

	cli := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(state.Context, "GET", "https://japi.rest/discord/v1/application/"+clientId, nil)

	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err.Error())
	}

	japiKey := state.Config.JAPI.Key
	if japiKey != "" {
		req.Header.Set("Authorization", japiKey)
	}

	resp, err := cli.Do(req)

	if err != nil {
		return nil, fmt.Errorf("error making request: %s", err.Error())
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("we're being ratelimited by our anti-abuse provider! Please try again in %s seconds", resp.Header.Get("Retry-After"))
	} else if resp.StatusCode > 400 {
		return nil, fmt.Errorf("we couldn't find a bot with that client ID! Status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("we couldn't find a bot with that client ID! Status code: " + strconv.Itoa(resp.StatusCode))
	}

	var data japidata

	err = json.NewDecoder(resp.Body).Decode(&data)

	if err != nil {
		return nil, err
	}

	var metadata *DiscordBotMeta

	if data.Data.Message != "" {
		// Fallback to RPC, but this is less accurate
		req, err := http.NewRequestWithContext(state.Context, "GET", state.Config.Meta.PopplioProxy+"/api/v10/applications/"+clientId+"/rpc", nil)

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
			if cidInt > 132550911590400000 {
				return nil, errors.New("fallbackNeeded")
			}

			fallbackBotId = clientId
		}

		// Check that the client id is also a bot (or rather, hope)
		user, err := dovewing.GetDiscordUser(state.Context, fallbackBotId)

		if err != nil {
			return nil, errors.New("the client id provided is not an actual bot id")
		}

		metadata = &DiscordBotMeta{
			BotID:     fallbackBotId,
			ClientID:  clientId,
			Name:      user.Username,
			Avatar:    user.Avatar,
			BotPublic: rpcFallbackData.BotPublic,
			Fallback:  true,
		}
	} else {
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

		if data.Data.Bot == nil || data.Data.Application == nil {
			return nil, errors.New("woah there, we found an application with no associated bot?")
		}

		if data.Data.Bot.ID == "" {
			return nil, errors.New("woah there, we found an application with no associated bot?")
		}

		user, err := dovewing.GetDiscordUser(state.Context, data.Data.Bot.ID)

		if err != nil {
			return nil, errors.New("please contact support, an error has occured while trying to fetch basic info")
		}

		metadata = &DiscordBotMeta{
			BotID:       data.Data.Bot.ID,
			ClientID:    clientId,
			Name:        user.Username,
			GuildCount:  data.Data.Bot.ApproximateGuildCount,
			BotPublic:   data.Data.Application.BotPublic,
			Avatar:      user.Avatar,
			Flags:       data.Data.Bot.PublicFlagsArray,
			Description: data.Data.Application.Description,
			Tags:        data.Data.Application.Tags,
			Fallback:    false,
		}
	}

	// Check if the bot is already in the database
	var count int

	err = state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", metadata.BotID).Scan(&count)

	if err != nil {
		return nil, errors.New("failed to check if bot is already in the database")
	}

	if count > 0 {
		// Get bot type
		var listType string

		err = state.Pool.QueryRow(state.Context, "SELECT type FROM bots WHERE bot_id = $1", metadata.BotID).Scan(&listType)

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
