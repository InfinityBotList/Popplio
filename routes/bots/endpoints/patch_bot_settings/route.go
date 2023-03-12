package patch_bot_settings

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"reflect"
	"strconv"
	"strings"

	docs "github.com/infinitybotlist/doclib"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type BotSettingsUpdate struct {
	Short      string       `db:"short" json:"short" validate:"required,min=30,max=150" msg:"Short description must be between 30 and 150 characters"` // impld
	Long       string       `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                 // impld
	Prefix     string       `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`              // impld
	Invite     string       `db:"invite" json:"invite" validate:"required,https" msg:"Invite is required and must be a valid HTTPS URL"`               // impld
	Banner     *string      `db:"banner" json:"banner" validate:"omitempty,https" msg:"Background must be a valid HTTPS URL"`                          // impld
	Library    string       `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`           // impld
	ExtraLinks []types.Link `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                   // Impld
	Tags       []string     `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	NSFW       bool         `db:"nsfw" json:"nsfw"`
}

func updateBotsArgs(bot BotSettingsUpdate) []any {
	return []any{
		bot.Short,
		bot.Long,
		bot.Prefix,
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
	return &docs.Doc{
		Summary:     "Update Bot Settings",
		Description: "Updates a bots settings. Returns 204 on success",
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
		Req:  BotSettingsUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionEditBotSettings) {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit bot settings", Error: true},
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
	botUser, err := utils.GetDiscordUser(d.Context, id)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal Error: Failed to get bot user",
				Error:   true,
			},
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
	botArgs = append(botArgs, id)

	// Update the bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET "+updateSqlStr+" WHERE bot_id=$"+strconv.Itoa(len(botArgs)), botArgs...)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Clear cache
	utils.ClearBotCache(d.Context, id)

	// Send a message to the bot logs channel
	state.Discord.ChannelMessageSendComplex(state.Config.Channels.BotLogs, &discordgo.MessageSend{
		Content: "",
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend + "/bots/" + id,
				Title: "Bot Updated",
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: botUser.Avatar,
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Name",
						Value:  botUser.Username,
						Inline: true,
					},
					{
						Name:   "Bot ID",
						Value:  "<@" + id + ">",
						Inline: true,
					},
					{
						Name:   "User",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: true,
					},
				},
			},
		},
	})
	return api.DefaultResponse(http.StatusNoContent)
}
