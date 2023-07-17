package transfer_bot_to_team

import (
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

type AddBotTeam struct {
	TeamID string `json:"team_id" validate:"required"`
}

var (
	compiledMessages = uapi.CompileValidationErrors(AddBotTeam{})
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
		Req:  AddBotTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "bid")

	var payload AddBotTeam

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

	// Check linked main owner
	var linkedOwnerId pgtype.Text
	var teamOwner pgtype.Text

	err = state.Pool.QueryRow(d.Context, "SELECT owner FROM bots WHERE bot_id = $1", botId).Scan(&linkedOwnerId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = state.Pool.QueryRow(d.Context, "SELECT team_owner FROM bots WHERE bot_id = $1", botId).Scan(&teamOwner)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if teamOwner.Valid {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "This bot is already in a team"},
		}
	}

	if linkedOwnerId.Valid && linkedOwnerId.String != d.Auth.ID {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be the owner to transfer a bot to a team"},
		}
	}

	// Convert ID to UUID
	if !utils.IsValidUUID(payload.TeamID) {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Team ID must be a valid UUID"},
		}
	}

	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", payload.TeamID).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Team not found"},
		}
	}

	// Ensure manager is a member of the team
	var managerCount int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", payload.TeamID, d.Auth.ID).Scan(&managerCount)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if managerCount == 0 {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are not a member of this team"},
		}
	}

	var managerPerms []types.TeamPermission
	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", payload.TeamID, d.Auth.ID).Scan(&managerPerms)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	mp := teams.NewPermissionManager(managerPerms)

	if !mp.Has(teams.TeamPermissionAddNewBots) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to add new bots"},
		}
	}

	// Check if bot is already in the team
	if teamOwner.String == payload.TeamID {
		return uapi.HttpResponse{
			Status: http.StatusConflict,
			Json:   types.ApiError{Message: "This bot is already in the team"},
		}
	}

	// Transfer bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", payload.TeamID, botId)

	if err != nil {
		state.Logger.Error(err)
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
