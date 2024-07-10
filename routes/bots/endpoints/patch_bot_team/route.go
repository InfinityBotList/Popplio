package patch_bot_team

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Patch Bot Team",
		Description: `Transfers a bot to another team. 

Semantically equivalent to:
- Remove bot in question from list
- Readd bot to list with same data
- Transfer bot ownership to team

The below are the requirements for this due to the above:

- The user must have the "Delete Bots" permission in the team they are transferring the bot from
- The user must have the "Add New Bots" permission in the team they are transferring the bot to

The bots ownership will be transferred to the new team.

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
	id := chi.URLParam(r, "bid")

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

	err := api.AuthzEntityPermissionCheck(
		d.Context,
		d.Auth,
		api.TargetTypeBot,
		id,
		kittycat.Permission{Namespace: api.TargetTypeBot, Perm: teams.PermissionDelete},
	)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be able to delete the bot in the old team to transfer it: " + err.Error()},
		}
	}

	err = api.AuthzEntityPermissionCheck(
		d.Context,
		d.Auth,
		api.TargetTypeTeam,
		payload.TeamID,
		kittycat.Permission{Namespace: api.TargetTypeBot, Perm: teams.PermissionAdd},
	)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be able to add the bot in the new team to transfer it: " + err.Error()},
		}
	}

	// Get old team ID for audit log
	var currentBotTeam pgtype.UUID

	err = state.Pool.QueryRow(d.Context, "SELECT team_owner FROM bots WHERE bot_id = $1", id).Scan(&currentBotTeam)

	if err != nil {
		state.Logger.Error("Error getting current team for bot: ", zap.Error(err), zap.String("botID", id), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Transfer bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", payload.TeamID, id)

	if err != nil {
		state.Logger.Error("Error transferring bot to team", zap.String("botID", id), zap.String("userID", d.Auth.ID), zap.String("newTeamID", payload.TeamID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Send message to mod logs
	state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Parse() + "/bots/" + id,
				Title: "Bot Team Update!",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Bot ID",
						Value:  id,
						Inline: true,
					},
					{
						Name:   "Performed By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: true,
					},
					{
						Name:  "Old Team",
						Value: fmt.Sprintf("[View Team](%s/team/%s)", state.Config.Sites.Frontend.Parse(), validators.EncodeUUID(currentBotTeam.Bytes)),
					},
					{
						Name:  "New Team",
						Value: fmt.Sprintf("[View Team](%s/team/%s)", state.Config.Sites.Frontend.Parse(), payload.TeamID),
					},
				},
			},
		},
	})

	return uapi.DefaultResponse(http.StatusNoContent)
}
