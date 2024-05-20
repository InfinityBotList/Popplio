package patch_bot_settings

import (
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"reflect"
	"strconv"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

func updateBotsArgs(bot types.BotSettingsUpdate) []any {
	return []any{
		bot.Short,
		bot.Long,
		bot.Prefix,
		bot.Invite,
		bot.Library,
		bot.ExtraLinks,
		bot.Tags,
		bot.NSFW,
		bot.CaptchaOptOut,
	}
}

var (
	compiledMessages = uapi.CompileValidationErrors(types.BotSettingsUpdate{})
	updateSql        = []string{}
	updateSqlStr     string
)

func Setup() {
	// Creates the updateSql
	for i, field := range reflect.VisibleFields(reflect.TypeOf(types.BotSettingsUpdate{})) {
		if field.Tag.Get("db") != "" {
			updateSql = append(updateSql, field.Tag.Get("db")+"=$"+strconv.Itoa(i+1))
		}
	}

	updateSqlStr = strings.Join(updateSql, ",")
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Bot Settings",
		Description: "Updates a bots settings. You must have 'Edit Bot Settings' in the team if the bot is in a team. Returns 204 on success",
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
		Req:  types.BotSettingsUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "bid")

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", id)

	if err != nil {
		state.Logger.Error("Failed to get entity perms: ", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetType", "bot"), zap.String("targetID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: "bot", Perm: teams.PermissionEdit}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit bot settings"},
		}
	}

	// Read payload from body
	var payload types.BotSettingsUpdate

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	err = validators.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	// Get bot discord user
	botUser, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to get bot user: ", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", id))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Failed to get bot user:" + err.Error()},
		}
	}

	// Update the bot
	// Get the arguments to pass when adding the bot
	botArgs := updateBotsArgs(payload)

	if len(updateSql) != len(botArgs) {
		state.Logger.Error("updateSql and botArgs do not match in length", zap.Any("updateSql", updateSql), zap.Any("botArgs", botArgs))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Internal Error: The number of columns and arguments do not match"},
		}
	}

	// Add the bot id to the end of the args
	botArgs = append(botArgs, id)

	// Update the bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET "+updateSqlStr+", updated_at = NOW() WHERE bot_id=$"+strconv.Itoa(len(botArgs)), botArgs...)

	if err != nil {
		state.Logger.Error("Failed to update bot: ", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Send a message to the bot logs channel
	state.Discord.ChannelMessageSendComplex(state.Config.Channels.BotLogs, &discordgo.MessageSend{
		Content: "",
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/bots/" + id,
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
	return uapi.DefaultResponse(http.StatusNoContent)
}
