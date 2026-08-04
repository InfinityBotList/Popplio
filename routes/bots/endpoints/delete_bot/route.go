// Package delete_bot implements DELETE /bots/{id} — "Delete Bot".
//
// Deletes a bot from the list. This is *irreversible*. You must have 'Delete
// Bots' in the team if the bot is in a team. Returns 204 on success.
package delete_bot

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
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
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	// Delete bot, arcadia will automatically cleanout generic entities associated with the bot in a controlled manner
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	defer tx.Rollback(d.Context)

	_, err = tx.Exec(d.Context, "DELETE FROM bots WHERE bot_id = $1", id)

	if err != nil {
		return resp.Err("Error while deleting bot", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	// Send embed to bot log channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: "",
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend.Parse() + "/bots/" + id,
				Title: "Bot Deleted",
				Color: 0xff0000,
				Fields: []discord.EmbedField{
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
