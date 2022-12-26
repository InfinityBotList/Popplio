package patch_bot_settings

import (
	"fmt"
	"net/http"
	"os"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"reflect"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slices"
)

type BotSettingsUpdate struct {
	Short            string       `db:"short" json:"short" validate:"required,min=50,max=100" msg:"Short description must be between 50 and 100 characters"`                                                                                                 // impld
	Long             string       `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                                                                                                                 // impld
	Prefix           string       `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`                                                                                                              // impld
	AdditionalOwners []string     `db:"additional_owners" json:"additional_owners" validate:"required,unique,max=7,dive,numeric" msg:"You can only have a maximum of 7 additional owners" amsg:"Each additional owner must be a valid snowflake and unique"` // impld
	Invite           string       `db:"invite" json:"invite" validate:"required,url" msg:"Invite is required and must be a valid URL"`                                                                                                                       // impld
	Banner           *string      `db:"banner" json:"banner" validate:"omitempty,url" msg:"Background must be a valid URL"`                                                                                                                                  // impld
	Library          string       `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`                                                                                                           // impld
	ExtraLinks       []types.Link `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                                                                                                                   // Impld
	Tags             []string     `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=20,alpha,notblank,nonvulgar,nospaces" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 20 characters and alphabetic"`
	NSFW             bool         `db:"nsfw" json:"nsfw"`
}

func updateBotsArgs(bot BotSettingsUpdate) []any {
	return []any{
		bot.Short,
		bot.Long,
		bot.Prefix,
		bot.AdditionalOwners,
		bot.Invite,
		bot.Banner,
		bot.Library,
		bot.ExtraLinks,
		bot.Tags,
		bot.NSFW,
	}
}

var (
	compiledMessages = api.CompileValidationErrors(BotSettingsUpdate{})
	updateSql        = []string{}
	updateSqlStr     string
)

func Setup() {
	// Creates the updateSql
	for i, field := range reflect.VisibleFields(reflect.TypeOf(BotSettingsUpdate{})) {
		if field.Tag.Get("db") != "" {
			updateSql = append(updateSql, field.Tag.Get("db")+"=$"+strconv.Itoa(i+1))
		}
	}

	updateSqlStr = strings.Join(updateSql, ",")
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "PATCH",
		Path:        "/users/{uid}/bots/{bid}/settings",
		OpId:        "patch_bot_settings",
		Summary:     "Update Bot Settings",
		Description: "Updates a bots vanity. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:      BotSettingsUpdate{},
		Resp:     types.ApiError{},
		Tags:     []string{api.CurrentTag},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	botIdParam := chi.URLParam(r, "bid")

	// Resolve id
	var botId string

	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE "+constants.ResolveBotSQL, botIdParam).Scan(&botId)

	if err != nil {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Validate that they actually own this bot
	isOwner, err := utils.IsBotOwner(d.Context, d.Auth.ID, botId)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	if !isOwner {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You do not own the bot you are trying to manage", Error: true},
		}
	}

	// Read payload from body
	var payload BotSettingsUpdate

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	if !strings.HasPrefix(payload.Invite, "https://") {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invite must start with https://",
				Error:   true,
			},
		}
	}

	if payload.Banner != nil && !strings.HasPrefix(*payload.Banner, "https://") {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Background/Banner URL must start with https://",
				Error:   true,
			},
		}
	}

	if slices.Contains(payload.AdditionalOwners, d.Auth.ID) {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "You cannot be an additional owner",
				Error:   true,
			},
		}
	}

	if slices.Contains(payload.Tags, "nsfw") && !payload.NSFW {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "You cannot add the nsfw tag without setting nsfw to true",
				Error:   true,
			},
		}
	}

	err = utils.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: err.Error(),
				Error:   true,
			},
		}
	}

	// Get bot discord user
	botUser, err := utils.GetDiscordUser(botId)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal Error: Failed to get bot user",
				Error:   true,
			},
		}
	}

	// Ensure the additional owners exist
	for _, owner := range payload.AdditionalOwners {
		ownerObj, err := utils.GetDiscordUser(owner)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "One of the additional owners of this bot does not exist [" + owner + "]: " + err.Error(),
					Error:   true,
				},
			}
		}

		if ownerObj.Bot {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "One of the additional owners of this bot is actually a bot [" + owner + "]",
					Error:   true,
				},
			}
		}
	}

	// Update the bot

	// Get the arguments to pass when adding the bot
	botArgs := updateBotsArgs(payload)

	if len(updateSql) != len(botArgs) {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal Error: The number of columns and arguments do not match",
				Error:   true,
			},
		}
	}

	// Add the bot id to the end of the args
	botArgs = append(botArgs, botId)

	// Update the bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET "+updateSqlStr+" WHERE bot_id=$"+strconv.Itoa(len(botArgs)), botArgs...)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Clear cache
	utils.ClearBotCache(d.Context, botId)

	notifications.MessageNotifyChannel <- notifications.DiscordLog{
		ChannelID: os.Getenv("BOT_LOGS_CHANNEL"),
		Message: &discordgo.MessageSend{
			Content: "",
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:   os.Getenv("FRONTEND_URL") + "/bots/" + botId,
					Title: "Bot Updated",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Name",
							Value: botUser.Username,
						},
						{
							Name:  "Bot ID",
							Value: "<@" + botId + ">",
						},
						{
							Name:  "User",
							Value: fmt.Sprintf("<@%s>", d.Auth.ID),
						},
						{
							Name: "Additional Owners",
							Value: func() string {
								if len(payload.AdditionalOwners) == 0 {
									return "None"
								}

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

	return api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
