package patch_bot_team

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

type PatchBotTeam struct {
	TeamID string `json:"team_id" validate:"required"`
}

var (
	compiledMessages = uapi.CompileValidationErrors(PatchBotTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Patch Bot Team",
		Description: `Transfers a bot owned by a team to another team. 

Semantically equivalent to:
- Remove bot in question from list
- Readd bot to list with same data
- Transfer bot ownership to team

The below are the requirements for this due to the above:

- The user must have the "Delete Bots" permission in the team they are transferring the bot from
- The user must have the "Add New Bots" permission in the team they are transferring the bot to

The bots ownership will be transferred to to the team.

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
		Req:  PatchBotTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botName := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(d.Context, botName)

	if err != nil {
		state.Logger.Error("Resolve Error", err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var payload PatchBotTeam

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

	// Get current team of bot
	var currentBotTeam pgtype.UUID

	err = state.Pool.QueryRow(d.Context, "SELECT team_owner FROM bots WHERE bot_id = $1", id).Scan(&currentBotTeam)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !currentBotTeam.Valid {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "This bot is not in a team?"},
		}
	}

	// Check if manager
	// Ensure manager is a member of the team
	var managerCount int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", currentBotTeam, d.Auth.ID).Scan(&managerCount)

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

	// Get the manager's permissions in current team
	var managerPerms []types.TeamPermission
	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", currentBotTeam, d.Auth.ID).Scan(&managerPerms)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Convert ID to UUID
	if !utils.IsValidUUID(payload.TeamID) {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Team ID must be a valid UUID"},
		}
	}

	// Find new team
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

	// Get manager perms in new team
	var newTeamPerms []types.TeamPermission

	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", payload.TeamID, d.Auth.ID).Scan(&newTeamPerms)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !teams.NewPermissionManager(managerPerms).Has(teams.TeamPermissionDeleteBots) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be able to delete bots on the current team"},
		}
	}

	if !teams.NewPermissionManager(newTeamPerms).Has(teams.TeamPermissionAddNewBots) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You must be able to add new bots on the new team"},
		}
	}

	// Transfer bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", payload.TeamID, id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Clear cache
	utils.ClearBotCache(d.Context, id)

	// Send message to mod logs
	state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/bots/" + id,
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
						Value: fmt.Sprintf("[View Team](%s/team/%s)", state.Config.Sites.Frontend, encodeUUID(currentBotTeam.Bytes)),
					},
					{
						Name:  "New Team",
						Value: fmt.Sprintf("[View Team](%s/team/%s)", state.Config.Sites.Frontend, payload.TeamID),
					},
				},
			},
		},
	})

	return uapi.DefaultResponse(http.StatusNoContent)
}

func encodeUUID(src [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}
