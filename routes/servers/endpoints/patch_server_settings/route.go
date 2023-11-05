package patch_server_settings

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
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

func updateServerArgs(server types.ServerSettingsUpdate) []any {
	return []any{
		server.Short,
		server.Long,
		server.ExtraLinks,
		server.Tags,
		server.NSFW,
		server.CaptchaOptOut,
	}
}

var (
	compiledMessages = uapi.CompileValidationErrors(types.ServerSettingsUpdate{})
	updateSql        = []string{}
	updateSqlStr     string
)

func Setup() {
	// Creates the updateSql
	for i, field := range reflect.VisibleFields(reflect.TypeOf(types.ServerSettingsUpdate{})) {
		if field.Tag.Get("db") != "" {
			updateSql = append(updateSql, field.Tag.Get("db")+"=$"+strconv.Itoa(i+1))
		}
	}

	updateSqlStr = strings.Join(updateSql, ",")
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Server Settings",
		Description: "Updates a servers settings. You must have 'Edit Server Settings' in the team if the bot is in a team. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "sid",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.ServerSettingsUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "sid")

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "server", id)

	if err != nil {
		state.Logger.Error("Error while getting entity perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetType", "server"), zap.String("targetID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has("server", teams.PermissionEdit) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit server settings"},
		}
	}

	// Read payload from body
	var payload types.ServerSettingsUpdate

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payloa
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

	// Update the bot
	// Get the arguments to pass when adding the bot
	serverArgs := updateServerArgs(payload)

	if len(updateSql) != len(serverArgs) {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Internal Error: The number of columns and arguments do not match"},
		}
	}

	// Add the bot id to the end of the args
	serverArgs = append(serverArgs, id)

	// Update the bot
	_, err = state.Pool.Exec(d.Context, "UPDATE servers SET "+updateSqlStr+" WHERE server_id=$"+strconv.Itoa(len(serverArgs)), serverArgs...)

	if err != nil {
		state.Logger.Error("Error while updating server", zap.Error(err), zap.String("serverID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var name, avatar string

	err = state.Pool.QueryRow(d.Context, "SELECT name, avatar FROM servers WHERE server_id = $1", id).Scan(&name, &avatar)

	if err != nil {
		state.Logger.Error("Error while getting server info", zap.Error(err), zap.String("serverID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Send a message to the bot logs channel
	state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
		Content: "",
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/servers/" + id,
				Title: "Server Updated",
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: avatar,
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Name",
						Value:  name,
						Inline: true,
					},
					{
						Name:   "Server ID",
						Value:  id,
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
