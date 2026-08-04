// Package add_server implements PUT /servers — "Add Server".
//
// Adds a server to the database from an invite link. Resolves the guild via
// the invite (the tracking bot does not need to already be in the server).
// Returns 204 on success
package add_server

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"regexp"
	"slices"
	"strings"
	"time"

	"popplio/db"
	"popplio/perms"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/go-playground/validator/v10"
)

// createServerArgs must list values in the exact same order as
// db.GetCols(types.CreateServer{}) (i.e. types.CreateServer's field
// declaration order) — createServerCols/createServerParams build the SQL
// column list from that reflection order, and args are bound to columns
// positionally, so any drift between the two silently writes every field
// into the wrong column instead of failing loudly.
func createServerArgs(server types.CreateServer) []any {
	return []any{
		server.Invite,
		server.Short,
		server.Long,
		server.ExtraLinks,
		server.Tags,
		server.NSFW,
		server.TeamOwner,
		server.ServerID,
		server.Name,
		server.Avatar,
		server.TotalMembers,
		server.OnlineMembers,
		server.VanityRef,
	}
}

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateServer{})

	createServerColsArr = db.GetCols(types.CreateServer{})
	createServerCols    = strings.Join(createServerColsArr, ", ")

	createServerParams string
)

func Setup() {
	var paramsList = make([]string, len(createServerColsArr))
	for i := range createServerColsArr {
		paramsList[i] = fmt.Sprintf("$%d", i+1)
	}
	createServerParams = strings.Join(paramsList, ",")
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Add Server",
		Description: "Adds a server to the database from an invite link. Resolves the guild via the invite (the tracking bot does not need to already be in the server). Returns 204 on success",
		Req:         types.CreateServer{},
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "add_server",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error calculating ratelimits", err, zap.String("userID", d.Auth.ID))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var payload types.CreateServer

	hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	err = validators.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	invite, err := assets.ResolveInvite(d.Context, payload.Invite)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	payload.ServerID = invite.Guild.ID.String()
	payload.Name = invite.Guild.Name
	payload.Avatar = assets.GuildIconURL(invite.Guild)
	payload.TotalMembers = invite.ApproximateMemberCount
	payload.OnlineMembers = invite.ApproximatePresenceCount

	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", payload.ServerID).Scan(&count)

	if err != nil {
		return resp.Err("Error while checking if server is already in database", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	if count > 0 {
		return resp.Conflict("This server is already in the database")
	}

	vanity := strings.ReplaceAll(strings.ToLower(payload.Name), " ", "-")
	vanity = regexp.MustCompile("[^a-zA-Z0-9-]").ReplaceAllString(vanity, "")
	vanity = strings.TrimSuffix(vanity, "-")

	var vanityCount int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE code = $1", vanity).Scan(&vanityCount)

	if err != nil {
		return resp.Err("Error while checking if calculated vanity is already taken", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
	}

	if vanityCount > 0 {
		vanity = vanity + "-" + crypto.RandString(8)
	}

	systems, err := validators.GetWordBlacklistSystems(d.Context, vanity)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return resp.BadRequest("Error while getting word blacklist systems: " + err.Error())
	}

	if slices.Contains(systems, "vanity.code") {
		return resp.BadRequest("The chosen vanity is blacklisted")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	defer tx.Rollback(d.Context)

	// Check team owner here, to avoid a race condition
	if payload.TeamOwner != "" {
		entityPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", payload.TeamOwner)

		if err != nil {
			state.Logger.Error("Error while getting team perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("teamID", payload.TeamOwner), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
			return resp.BadRequest("Error getting user perms: " + err.Error())
		}

		if !entityPerms.Has(perms.EntityAddServers) {
			return resp.Forbidden("You do not have permission to add new servers to this team")
		}
	} else {
		var teamId = uuid.New()

		var vanityRef string
		err = tx.QueryRow(d.Context, "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'team', $2) RETURNING itag", teamId, payload.Name+crypto.RandString(16)).Scan(&vanityRef)

		if err != nil {
			return resp.BadRequest("Error while creating vanity: " + err.Error())
		}

		_, err = tx.Exec(d.Context, "INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'api/add_server')", teamId, payload.Name, vanityRef)

		if err != nil {
			return resp.BadRequest("Error while creating team: " + err.Error())
		}

		_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'api/add_server')", teamId, d.Auth.ID, []string{string(perms.EntityOwner)})

		if err != nil {
			return resp.BadRequest("Error while adding team member: " + err.Error())
		}

		payload.TeamOwner = teamId.String()
	}

	var itag pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, $3) RETURNING itag", vanity, payload.ServerID, "server").Scan(&itag)

	if err != nil {
		return resp.Err("Error while inserting vanity", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
	}

	payload.VanityRef = itag

	serverArgs := createServerArgs(payload)

	if len(createServerColsArr) != len(serverArgs) {
		return resp.ErrBody("createServerColsArr and serverArgs do not match in length", "Internal Error: The number of columns and arguments do not match", nil, zap.Any("createServerColsArr", createServerColsArr), zap.Any("serverArgs", serverArgs))
	}

	_, err = tx.Exec(d.Context, "INSERT INTO servers ("+createServerCols+") VALUES ("+createServerParams+")", serverArgs...)

	if err != nil {
		return resp.Err("Error while inserting server", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend.Parse() + "/servers/" + payload.ServerID,
				Title: "New Server Added",
				Fields: []discord.EmbedField{
					{
						Name:   "Name",
						Value:  payload.Name,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Server ID",
						Value:  payload.ServerID,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Added by",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: validators.TruePtr,
					},
				},
			},
		},
	})

	if err != nil {
		state.Logger.Error("Error while sending server logs message", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
