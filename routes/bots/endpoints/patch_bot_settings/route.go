// Package patch_bot_settings implements PATCH /bots/{id}/settings — "Update
// Bot Settings".
//
// Updates a bots settings. You must have 'Edit Bot Settings' in the team if
// the bot is in a team. Returns 204 on success
package patch_bot_settings

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"reflect"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

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
				Name:        "id",
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
	id := chi.URLParam(r, "id")

	// Read payload from body
	var payload types.BotSettingsUpdate

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	err = validators.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	// Get bot discord user
	botUser, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.ErrBody("Failed to get bot user: ", "Failed to get bot user.", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	// Update the bot
	// Get the arguments to pass when adding the bot
	botArgs := updateBotsArgs(payload)

	if len(updateSql) != len(botArgs) {
		return resp.ErrBody("updateSql and botArgs do not match in length", "Internal Error: The number of columns and arguments do not match", nil, zap.Any("updateSql", updateSql), zap.Any("botArgs", botArgs))
	}

	// Add the bot id to the end of the args
	botArgs = append(botArgs, id)

	// Update the bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET "+updateSqlStr+", updated_at = NOW() WHERE bot_id=$"+strconv.Itoa(len(botArgs)), botArgs...)

	if err != nil {
		return resp.Err("Failed to update bot: ", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	// Send a message to the bot logs channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
		Content: "",
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/bots/" + id,
				Title: "Bot Updated",
				Thumbnail: &discord.EmbedResource{
					URL: botUser.Avatar,
				},
				Fields: []discord.EmbedField{
					{
						Name:   "Name",
						Value:  botUser.Username,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Bot ID",
						Value:  "<@" + id + ">",
						Inline: validators.TruePtr,
					},
					{
						Name:   "User",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: validators.TruePtr,
					},
				},
			},
		},
	})

	if err != nil {
		return resp.ErrBody("Error while sending embed to mod logs channel", "Internal Error: While bot update was successful, an error occurred while sending the update embed to the mod logs channel", err, zap.String("serverID", id))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
