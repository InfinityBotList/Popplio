package add_bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type CreateBot struct {
	BotID            string       `json:"bot_id" validate:"required,numeric" msg:"Bot ID must be numeric"`
	ClientID         string       `json:"client_id" validate:"required,numeric" msg:"Client ID must be numeric"`
	Short            string       `json:"short" validate:"required,min=50,max=100" msg:"Short description must be between 50 and 100 characters"`
	Long             string       `json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`
	Prefix           string       `json:"prefix" validate:"required,min=1,max=10,alphanum" msg:"Prefix must be between 1 and 10 characters"`
	AdditionalOwners []string     `json:"additional_owners" validate:"required,max=7,dive,numeric" msg:"Additional owners must be numeric"`
	Invite           string       `json:"invite" validate:"required,url" msg:"Invite is required and must be a valid URL"`
	Background       *string      `json:"background" validate:"omitempty,url" msg:"Background must be a valid URL"`
	Library          string       `json:"library" validate:"required,min=1,max=50,alpha" msg:"Library must be between 1 and 50 characters"`
	ExtraLinks       []types.Link `json:"extra_links" validate:"required" msg:"Extra links must be sent"`
	Tags             []string     `json:"tags" validate:"required,min=1,max=5,dive,min=3,max=20,alpha" msg:"There must be between 1 and 5 tags" amsg:"Each tag must be between 3 and 20 characters"`
	NSFW             bool         `json:"nsfw" validate:"required" msg:"NSFW must be sent"`
	CrossAdd         bool         `json:"cross_add" validate:"required" msg:"Cross add must be sent"`
	StaffNote        *string      `json:"staff_note" validate:"required,max=1000" msg:"Staff note must be sent and must be less than 1000 characters"`
}

var compiledMessages = api.CompileValidationErrors(CreateBot{})

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PUT",
		Path:        "/bots",
		OpId:        "get_all_bots",
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list. Returns a ``Index`` object",
		Tags:        []string{api.CurrentTag},
		Resp:        CreateBot{},
	})
}

type Japidata struct {
	Cached bool `json:"cached"`
	Data   struct {
		Application struct {
			ID        string `json:"id"`
			BotPublic bool   `json:"bot_public"`
		} `json:"application"`
		Bot struct {
			ID                    string `json:"id"`
			ApproximateGuildCount int    `json:"approximate_guild_count"`
		} `json:"bot"`
	} `json:"data"`
}

// Represents a response from checkBotClientId
type checkBotClientIdResp struct {
	guildCount int
}

func (bot *CreateBot) checkBotClientId() (*checkBotClientIdResp, error) {
	cli := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://japi.rest/discord/v1/application/"+bot.ClientID, nil)

	if err != nil {
		return nil, err
	}

	japiKey := os.Getenv("JAPI_KEY")
	if japiKey != "" {
		req.Header.Set("Authorization", os.Getenv("JAPI_KEY"))
	}

	resp, err := cli.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("we're being ratelimited by our anti-abuse provider! Please try again in %s seconds", resp.Header.Get("Retry-After"))
	} else if resp.StatusCode > 400 {
		return nil, fmt.Errorf("we couldn't find a bot with that client ID! Status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	var data Japidata

	err = json.NewDecoder(resp.Body).Decode(&data)

	if err != nil {
		return nil, err
	}

	if !data.Data.Application.BotPublic {
		return nil, fmt.Errorf("bot is not public")
	}

	if !data.Cached {
		state.Logger.With(
			zap.String("bot_id", bot.BotID),
			zap.String("client_id", bot.ClientID),
		).Info("JAPI cache MISS")
	} else {
		state.Logger.With(
			zap.String("bot_id", bot.BotID),
			zap.String("client_id", bot.ClientID),
		).Info("JAPI cache HIT")
	}

	if bot.BotID != data.Data.Bot.ID || bot.ClientID != data.Data.Application.ID {
		return nil, fmt.Errorf("the bot ID provided does not match the bot ID found")
	}

	return &checkBotClientIdResp{
		guildCount: data.Data.Bot.ApproximateGuildCount,
	}, nil
}

func Route(d api.RouteData, r *http.Request) {
	defer r.Body.Close()

	var payload CreateBot

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if len(bodyBytes) == 0 {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "A body is required for this endpoint",
				Error:   true,
			},
		}
		return
	}

	err = json.Unmarshal(bodyBytes, &payload)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid JSON: " + err.Error(),
				Error:   true,
			},
		}
		return
	}

	// Validate the payload
	validate := validator.New()

	err = validate.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		d.Resp <- api.ValidatorErrorResponse(compiledMessages, errors)

		return
	}

	if !strings.HasPrefix(payload.Invite, "https://") {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invite must start with https://",
				Error:   true,
			},
		}
		return
	}

	if payload.Background != nil && !strings.HasPrefix(*payload.Background, "https://") {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Background must start with https://",
				Error:   true,
			},
		}
		return
	}

	if slices.Contains(payload.AdditionalOwners, d.Auth.ID) {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "You cannot be an additional owner",
				Error:   true,
			},
		}
		return
	}

	if slices.Contains(payload.Tags, "nsfw") && !payload.NSFW {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "You cannot add the nsfw tag without setting nsfw to true",
				Error:   true,
			},
		}
		return
	}

	err = utils.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: err.Error(),
				Error:   true,
			},
		}
		return
	}

	_, err = payload.checkBotClientId()

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Hmmm..." + err.Error(),
				Error:   true,
			},
		}
		return
	}
}
