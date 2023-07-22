package delete_bot

import (
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
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
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(d.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionDeleteBots) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to delete this bot"},
		}
	}

	// Clear cache
	utils.ClearBotCache(d.Context, id)

	// Delete bot
	tx, err := state.Pool.Begin(d.Context)

	defer tx.Rollback(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = tx.Exec(d.Context, "DELETE FROM bots WHERE bot_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Delete generic entities
	for _, table := range []string{"reviews", "webhook_logs"} {
		_, err = tx.Exec(d.Context, "DELETE FROM "+table+" WHERE target_id = $1 AND target_type = 'bot'", id)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
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
