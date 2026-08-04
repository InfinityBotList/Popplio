// Package patch_bot_team implements PATCH /users/{uid}/bots/{bid}/teams —
// "Patch Bot Team".
//
// Transfers a bot to another team.
package patch_bot_team

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/api/resp"
	"popplio/perms"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

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
		return resp.BadRequest("Team ID must be provided")
	}

	err := api.AuthzEntityPermissionCheck(
		d.Context,
		d.Auth,
		api.TargetTypeBot,
		id,
		perms.EntityDeleteBots,
	)

	if err != nil {
		return resp.Forbidden("You must be able to delete the bot in the old team to transfer it: " + err.Error())
	}

	err = api.AuthzEntityPermissionCheck(
		d.Context,
		d.Auth,
		api.TargetTypeTeam,
		payload.TeamID,
		perms.EntityAddBots,
	)

	if err != nil {
		return resp.Forbidden("You must be able to add the bot in the new team to transfer it: " + err.Error())
	}

	// Get old team ID for audit log
	var currentBotTeam pgtype.UUID

	err = state.Pool.QueryRow(d.Context, "SELECT team_owner FROM bots WHERE bot_id = $1", id).Scan(&currentBotTeam)

	if err != nil {
		return resp.Err("Error getting current team for bot: ", err, zap.String("botID", id), zap.String("userID", d.Auth.ID))
	}

	// Transfer bot
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", payload.TeamID, id)

	if err != nil {
		return resp.Err("Error transferring bot to team", nil, zap.String("botID", id), zap.String("userID", d.Auth.ID), zap.String("newTeamID", payload.TeamID))
	}

	// Send message to mod logs
	state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend.Parse() + "/bots/" + id,
				Title: "Bot Team Update!",
				Fields: []discord.EmbedField{
					{
						Name:   "Bot ID",
						Value:  id,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Performed By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: validators.TruePtr,
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
