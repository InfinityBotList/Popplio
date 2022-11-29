package add_bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type CreateBot struct {
	BotID            string       `db:"bot_id" json:"bot_id" validate:"required,numeric" msg:"Bot ID must be numeric"`                                                                                                                                       // impld
	ClientID         string       `db:"client_id" json:"client_id" validate:"required,numeric" msg:"Client ID must be numeric"`                                                                                                                              // impld
	Short            string       `db:"short" json:"short" validate:"required,min=50,max=100" msg:"Short description must be between 50 and 100 characters"`                                                                                                 // impld
	Long             string       `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                                                                                                                 // impld
	Prefix           string       `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`                                                                                                              // impld
	AdditionalOwners []string     `db:"additional_owners" json:"additional_owners" validate:"required,unique,max=7,dive,numeric" msg:"You can only have a maximum of 7 additional owners" amsg:"Each additional owner must be a valid snowflake and unique"` // impld
	Invite           string       `db:"invite" json:"invite" validate:"required,url" msg:"Invite is required and must be a valid URL"`                                                                                                                       // impld
	Background       *string      `db:"banner" json:"background" validate:"omitempty,url" msg:"Background must be a valid URL"`                                                                                                                              // impld
	Library          string       `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`                                                                                                           // impld
	ExtraLinks       []types.Link `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                                                                                                                   // Impld
	Tags             []string     `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=20,alpha,notblank,nonvulgar,nospaces" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 20 characters and alphabetic"`
	NSFW             bool         `db:"nsfw" json:"nsfw"`
	CrossAdd         bool         `db:"cross_add" json:"cross_add"`
	StaffNote        *string      `db:"approval_note" json:"staff_note" validate:"omitempty,max=512" msg:"Staff note must be less than 512 characters if sent"` // impld

	// Internal fields
	QueueName  *string `db:"queue_name" json:"-" validate:"omitempty,notpresent"`
	Owner      *string `db:"owner" json:"-" validate:"omitempty,notpresent"`
	Vanity     *string `db:"vanity" json:"-" validate:"omitempty,notpresent"`
	GuildCount *int    `db:"servers" json:"-" validate:"omitempty,notpresent"`
}

func createBotsArgs(bot CreateBot) []any {
	return []any{
		bot.BotID,
		bot.ClientID,
		bot.Short,
		bot.Long,
		bot.Prefix,
		bot.AdditionalOwners,
		bot.Invite,
		bot.Background,
		bot.Library,
		bot.ExtraLinks,
		bot.Tags,
		bot.NSFW,
		bot.CrossAdd,
		bot.StaffNote,
		bot.QueueName,
		bot.Owner,
		bot.Vanity,
		bot.GuildCount,
	}
}

var (
	compiledMessages = api.CompileValidationErrors(CreateBot{})

	createBotsColsArr = utils.GetCols(CreateBot{})
	createBotsCols    = strings.Join(createBotsColsArr, ", ")

	// $1, $2, $3, etc, using the length of the array
	createBotsParams string
)

func init() {
	var paramsList []string = make([]string, len(createBotsColsArr))
	for i := 0; i < len(createBotsColsArr); i++ {
		paramsList[i] = fmt.Sprintf("$%d", i+1)
	}

	createBotsParams = strings.Join(paramsList, ",")
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PUT",
		Path:        "/bots",
		OpId:        "add_bot",
		Summary:     "Create Bot",
		Description: "Adds a bot to the database. The main owner will be the user who created the bot. Returns 204 on success",
		Tags:        []string{api.CurrentTag},
		Req:         CreateBot{},
		Resp:        types.ApiError{},
		AuthType:    []types.TargetType{types.TargetTypeUser},
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
			Username              string `json:"username"`
		} `json:"bot"`
	} `json:"data"`
}

// Represents a response from checkBotClientId
type checkBotClientIdResp struct {
	guildCount int
	botName    string
}

func (bot *CreateBot) checkBotClientId(ctx context.Context) (*checkBotClientIdResp, error) {
	cli := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://japi.rest/discord/v1/application/"+bot.ClientID, nil)

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
		botName:    data.Data.Bot.Username,
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

	err = state.Validator.Struct(payload)

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

	// Check if the bot is already in the database
	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", payload.BotID).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if count > 0 {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This bot is already in the database",
				Error:   true,
			},
		}
		return
	}

	// Ensure the bot actually exists right now
	bot, err := utils.GetDiscordUser(payload.BotID)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This bot does not exist: " + err.Error(),
				Error:   true,
			},
		}
		return
	}

	if !bot.Bot {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This user is not a bot",
				Error:   true,
			},
		}
		return
	}

	// Ensure the main owner exists
	_, err = utils.GetDiscordUser(d.Auth.ID)

	if err != nil {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "The main owner of this bot somehow does not exist: " + err.Error(),
				Error:   true,
			},
		}
		return
	}

	// Ensure the additional owners exist
	for _, owner := range payload.AdditionalOwners {
		ownerObj, err := utils.GetDiscordUser(owner)

		if err != nil {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "One of the additional owners of this bot does not exist [" + owner + "]: " + err.Error(),
					Error:   true,
				},
			}
			return
		}

		if ownerObj.Bot {
			d.Resp <- api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "One of the additional owners of this bot is actually a bot [" + owner + "]",
					Error:   true,
				},
			}
			return
		}
	}

	resp, err := payload.checkBotClientId(d.Context)

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

	payload.QueueName = &resp.botName
	payload.Owner = &d.Auth.ID
	payload.GuildCount = &resp.guildCount

	if payload.StaffNote == nil {
		defNote := "No note!"
		payload.StaffNote = &defNote
	}

	// Create initial vanity URL by removing all unicode characters and replacing spaces with dashes
	vanity := strings.ReplaceAll(strings.ToLower(resp.botName), " ", "-")
	vanity = regexp.MustCompile("[^a-zA-Z0-9-]").ReplaceAllString(vanity, "")
	vanity = strings.TrimSuffix(vanity, "-")

	payload.Vanity = &vanity

	// Get the arguments to pass when adding the bot
	botArgs := createBotsArgs(payload)

	if len(createBotsColsArr) != len(botArgs) {
		d.Resp <- api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal Error: The number of columns and arguments do not match",
				Error:   true,
			},
		}
		return
	}

	// Save the bot to the database
	_, err = state.Pool.Exec(d.Context, "INSERT INTO bots ("+createBotsCols+") VALUES ("+createBotsParams+")", botArgs...)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	notifications.MessageNotifyChannel <- types.DiscordLog{
		ChannelID: os.Getenv("BOT_LOGS_CHANNEL"),
		Message: &discordgo.MessageSend{
			Content: "",
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:   os.Getenv("FRONTEND_URL") + "/bots/" + payload.BotID,
					Title: "New Bot Added",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Name",
							Value:  resp.botName,
							Inline: true,
						},
						{
							Name:   "Bot ID",
							Value:  payload.BotID,
							Inline: true,
						},
						{
							Name:  "Main Owner",
							Value: fmt.Sprintf("<@%s>", d.Auth.ID),
						},
						{
							Name: "Additional Owners",
							Value: func() string {
								var owners []string
								for _, owner := range payload.AdditionalOwners {
									owners = append(owners, fmt.Sprintf("<@%s>", owner))
								}
								return strings.Join(owners, ", ")
							}(),
						},
					},
				},
			},
		},
	}

	d.Resp <- api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
