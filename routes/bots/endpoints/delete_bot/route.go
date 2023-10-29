package delete_bot

import (
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Bot",
		Description: "Deletes a bot from the list. This is *irreversible*. You must have 'Delete Bots' in the team if the bot is in a team. Returns 204 on success.",
		Resp:        types.ApiError{},
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
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "bid")

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", id)

	if err != nil {
		state.Logger.Error("Error while getting entity perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetType", "bot"), zap.String("targetID", id))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has("bot", teams.PermissionDelete) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to delete this bot"},
		}
	}

	// Delete bot, arcadia will automatically cleanout generic entities associated with the bot in a controlled manner
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	_, err = tx.Exec(d.Context, "DELETE FROM bots WHERE bot_id = $1", id)

	if err != nil {
		state.Logger.Error("Error while deleting bot", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Send embed to bot log channel
	_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
		Content: "",
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/bots/" + id,
				Title: "Bot Deleted",
				Color: 0xff0000,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Bot ID",
						Value: id,
					},
					{
						Name:  "Deleter",
						Value: fmt.Sprintf("<@%s>", d.Auth.ID),
					},
				},
			},
		},
	})

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Successfully deleted bot [ :) ] but we couldn't send a log message [ :( ]",
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
