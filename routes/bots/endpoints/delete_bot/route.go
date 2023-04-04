package delete_bot

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/doclib"
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

	if !perms.Has(teams.TeamPermissionDeleteBots) {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to delete this bot", Error: true},
		}
	}

	// Clear cache
	utils.ClearBotCache(d.Context, id)

	// Delete bot
	_, err = state.Pool.Exec(d.Context, "DELETE FROM bots WHERE bot_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Send embed to bot log channel
	_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
		Content: "",
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend + "/bots/" + id,
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
		return api.HttpResponse{
			Status: http.StatusOK,
			Data:   "Successfully deleted bot [ :) ] but we couldn't send a log message [ :( ]",
		}
	}

	return api.DefaultResponse(http.StatusNoContent)
}
