package add_server

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"popplio/db"
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
	kittycat "github.com/infinitybotlist/kittycat/go"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/go-playground/validator/v10"
)

var inviteCodeRegex = regexp.MustCompile(`(?:discord(?:\.gg|\.com/invite|app\.com/invite)/)?([a-zA-Z0-9-]+)/?$`)

func createServerArgs(server types.CreateServer) []any {
	return []any{
		server.ServerID,
		server.Name,
		server.Short,
		server.Long,
		server.ExtraLinks,
		server.Tags,
		server.NSFW,
		server.Invite,
		server.TotalMembers,
		server.OnlineMembers,
		server.TeamOwner,
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
		state.Logger.Error("Error calculating ratelimits", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
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
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	match := inviteCodeRegex.FindStringSubmatch(strings.TrimSpace(payload.Invite))

	if len(match) < 2 || match[1] == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Could not find an invite code in that URL"},
		}
	}

	invite, err := state.Discord.Rest().GetInvite(match[1])

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "That invite is invalid, expired, or the server has revoked it: " + err.Error()},
		}
	}

	if invite.Guild == nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "That invite is not for a server"},
		}
	}

	payload.ServerID = invite.Guild.ID.String()
	payload.Name = invite.Guild.Name
	payload.TotalMembers = invite.ApproximateMemberCount
	payload.OnlineMembers = invite.ApproximatePresenceCount

	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", payload.ServerID).Scan(&count)

	if err != nil {
		state.Logger.Error("Error while checking if server is already in database", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count > 0 {
		return uapi.HttpResponse{
			Status: http.StatusConflict,
			Json:   types.ApiError{Message: "This server is already in the database"},
		}
	}

	vanity := strings.ReplaceAll(strings.ToLower(payload.Name), " ", "-")
	vanity = regexp.MustCompile("[^a-zA-Z0-9-]").ReplaceAllString(vanity, "")
	vanity = strings.TrimSuffix(vanity, "-")

	var vanityCount int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE code = $1", vanity).Scan(&vanityCount)

	if err != nil {
		state.Logger.Error("Error while checking if calculated vanity is already taken", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if vanityCount > 0 {
		vanity = vanity + "-" + crypto.RandString(8)
	}

	systems, err := validators.GetWordBlacklistSystems(d.Context, vanity)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error while getting word blacklist systems: " + err.Error()},
		}
	}

	if slices.Contains(systems, "vanity.code") {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "The chosen vanity is blacklisted"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Check team owner here, to avoid a race condition
	if payload.TeamOwner != "" {
		perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", payload.TeamOwner)

		if err != nil {
			state.Logger.Error("Error while getting team perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("teamID", payload.TeamOwner), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
			}
		}

		if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: "server", Perm: teams.PermissionAdd}) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to add new servers to this team"},
			}
		}
	} else {
		var teamId = uuid.New()

		var vanityRef string
		err = tx.QueryRow(d.Context, "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'team', $2) RETURNING itag", teamId, payload.Name+crypto.RandString(16)).Scan(&vanityRef)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error while creating vanity: " + err.Error()},
			}
		}

		_, err = tx.Exec(d.Context, "INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'api/add_server')", teamId, payload.Name, vanityRef)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error while creating team: " + err.Error()},
			}
		}

		_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'api/add_server')", teamId, d.Auth.ID, []string{"global.*"})

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error while adding team member: " + err.Error()},
			}
		}

		payload.TeamOwner = teamId.String()
	}

	var itag pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, $3) RETURNING itag", vanity, payload.ServerID, "server").Scan(&itag)

	if err != nil {
		state.Logger.Error("Error while inserting vanity", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	payload.VanityRef = itag

	serverArgs := createServerArgs(payload)

	if len(createServerColsArr) != len(serverArgs) {
		state.Logger.Error("createServerColsArr and serverArgs do not match in length", zap.Any("createServerColsArr", createServerColsArr), zap.Any("serverArgs", serverArgs))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Internal Error: The number of columns and arguments do not match"},
		}
	}

	_, err = tx.Exec(d.Context, "INSERT INTO servers ("+createServerCols+") VALUES ("+createServerParams+")", serverArgs...)

	if err != nil {
		state.Logger.Error("Error while inserting server", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/servers/" + payload.ServerID,
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
