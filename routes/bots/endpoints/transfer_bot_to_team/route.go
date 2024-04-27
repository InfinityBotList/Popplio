package transfer_bot_to_team

import (
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Transfer Bot To Team",
		Description: `Transfers a bot owned by a user. 

The below are the requirements for this:

- The bot must not be already owned by a team (see the "Patch Bot Team" endpoint for transferring a bot from a team to a team)
- The bot must be owned by the user making the request
- The user must have the "Add New Bots" permission in the team they are transferring the bot to

The bots ownership will be transferred to the team.

Returns a 204 on success`,
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
		Req:  types.PatchBotTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "bid")

	var payload types.PatchBotTeam

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if payload.TeamID == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Team ID must be provided"},
		}
	}

	// Check linked main owner
	var owner pgtype.Text
	err := state.Pool.QueryRow(d.Context, "SELECT owner FROM bots WHERE bot_id = $1", botId).Scan(&owner)

	if err != nil {
		state.Logger.Error("Error checking bot owner: ", zap.Error(err), zap.String("botID", botId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !owner.Valid {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "This bot is already in a team"},
		}
	}

	if owner.Valid && owner.String != d.Auth.ID {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be the owner to transfer a bot to a team"},
		}
	}

	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", payload.TeamID).Scan(&count)

	if err != nil {
		state.Logger.Error("Error checking team: ", zap.Error(err), zap.String("teamID", payload.TeamID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Team not found"},
		}
	}

	newTeamPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", payload.TeamID)

	if err != nil {
		state.Logger.Error("Error checking team perms: ", zap.Error(err), zap.String("teamID", payload.TeamID))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	if !kittycat.HasPerm(newTeamPerms, kittycat.Build("bot", teams.PermissionAdd)) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be able to add the bot in the new team to transfer it"},
		}
	}

	// Transfer bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", payload.TeamID, botId)

	if err != nil {
		state.Logger.Error("Error transferring bot: ", zap.Error(err), zap.String("botID", botId), zap.String("teamID", payload.TeamID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Send message to mod logs
	state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/bots/" + botId,
				Title: "Bot Moved To Team (please audit!)",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Team",
						Value:  fmt.Sprintf("[View Team](%s/team/%s)", state.Config.Sites.Frontend.Production(), payload.TeamID),
						Inline: true,
					},
					{
						Name:   "Bot ID",
						Value:  botId,
						Inline: true,
					},
					{
						Name:   "Performed By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: true,
					},
				},
			},
		},
	})

	return uapi.DefaultResponse(http.StatusNoContent)
}
