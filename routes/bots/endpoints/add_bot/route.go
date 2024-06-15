package add_bot

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"popplio/api"
	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
)

type internalData struct {
	Owner      string
	GuildCount *int
	VanityRef  pgtype.UUID
}

func createBotsArgs(bot types.CreateBot, id internalData) []any {
	return []any{
		bot.BotID,
		bot.ClientID,
		bot.Short,
		bot.Long,
		bot.Prefix,
		bot.Invite,
		bot.Library,
		bot.ExtraLinks,
		bot.Tags,
		bot.NSFW,
		bot.StaffNote,
		id.Owner,
		id.GuildCount,
		id.VanityRef,
	}
}

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateBot{})

	createBotsColsArr = db.GetCols(types.CreateBot{})
	createBotsCols    = strings.Join(createBotsColsArr, ", ")

	// $1, $2, $3, etc, using the length of the array
	createBotsParams string
)

func Setup() {
	var paramsList []string = make([]string, len(createBotsColsArr))
	for i := 0; i < len(createBotsColsArr); i++ {
		paramsList[i] = fmt.Sprintf("$%d", i+1)
	}

	createBotsParams = strings.Join(paramsList, ",")
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Add Bot",
		Description: "Adds a bot to the database. Returns 204 on success",
		Req:         types.CreateBot{},
		Resp:        types.ApiError{},
		Params:      []docs.Parameter{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "add_bot",
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

	var payload types.CreateBot

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

	err = validators.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	// Check if the bot is already in the database
	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", payload.BotID).Scan(&count)

	if err != nil {
		state.Logger.Error("Error while checking if bot is already in database", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count > 0 {
		return uapi.HttpResponse{
			Status: http.StatusConflict,
			Json:   types.ApiError{Message: "This bot is already in the database"},
		}
	}

	// Ensure the bot actually exists right now
	bot, err := dovewing.GetUser(d.Context, payload.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "This bot does not exist: " + err.Error()},
		}
	}

	if !bot.Bot {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "This user is not a bot"},
		}
	}

	metadata, err := assets.CheckBot(d.Context, payload.BotID, payload.ClientID)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	if metadata.BotID != payload.BotID {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "The bot ID provided does not match the bot ID found"},
		}
	}

	if metadata.ListType != "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "This bot is already in the database"},
		}
	}

	if !metadata.BotPublic {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Bot is not public"},
		}
	}

	id := internalData{}

	if d.Auth.TargetType == api.TargetTypeTeam {
		payload.TeamOwner = d.Auth.ID
	}

	id.Owner = d.Auth.ID
	id.GuildCount = &metadata.GuildCount

	if payload.StaffNote == nil {
		defNote := "No note!"
		payload.StaffNote = &defNote
	}

	// Create initial vanity URL by removing all unicode characters and replacing spaces with dashes
	vanity := strings.ReplaceAll(strings.ToLower(metadata.Name), " ", "-")
	vanity = regexp.MustCompile("[^a-zA-Z0-9-]").ReplaceAllString(vanity, "")
	vanity = strings.TrimSuffix(vanity, "-")

	// Check that vanity isnt already taken
	var vanityCount int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE code = $1", vanity).Scan(&vanityCount)

	if err != nil {
		state.Logger.Error("Error while checking if calculated vanity is already taken", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if vanityCount > 0 {
		vanity = vanity + "-" + crypto.RandString(8)
	}

	// Save the bot to the database
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Create vanity
	var itag pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, $3) RETURNING itag", vanity, payload.BotID, "bot").Scan(&itag)

	if err != nil {
		state.Logger.Error("Error while inserting vanity", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	id.VanityRef = itag

	// Get the arguments to pass when adding the bot
	botArgs := createBotsArgs(payload, id)

	if len(createBotsColsArr) != len(botArgs) {
		state.Logger.Error("createBotsColsArr and botArgs do not match in length", zap.Any("createBotsColsArr", createBotsColsArr), zap.Any("botArgs", botArgs))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Internal Error: The number of columns and arguments do not match"},
		}
	}

	_, err = tx.Exec(d.Context, "INSERT INTO bots ("+createBotsCols+") VALUES ("+createBotsParams+")", botArgs...)

	if err != nil {
		state.Logger.Error("Error while inserting bot", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Check team owner here, to avoid a race condition
	if payload.TeamOwner != "" {
		perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", payload.TeamOwner)

		if err != nil {
			state.Logger.Error("Error while getting team perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("teamID", payload.TeamOwner), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
			}
		}

		if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: "bot", Perm: teams.PermissionAdd}) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to add new bots to this team"},
			}
		}

		_, err = tx.Exec(d.Context, "UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2", payload.TeamOwner, payload.BotID)

		if err != nil {
			state.Logger.Error("Error while updating bot team owner", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("teamID", payload.TeamOwner), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.BotLogs, &discordgo.MessageSend{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend.Production() + "/bots/" + payload.BotID,
				Title: "New Bot Added",
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: metadata.Avatar,
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Name",
						Value:  metadata.Name,
						Inline: true,
					},
					{
						Name:   "Bot ID",
						Value:  payload.BotID,
						Inline: true,
					},
					{
						Name: "Owner",
						Value: func() string {
							if payload.TeamOwner != "" {
								return fmt.Sprintf("[Team %s](%s/teams/%s)", payload.TeamOwner, state.Config.Sites.Frontend.Parse(), payload.TeamOwner)
							}
							return fmt.Sprintf("<@%s>", d.Auth.ID)
						}(),
						Inline: true,
					},
				},
			},
		},
	})

	if err != nil {
		state.Logger.Error("Error while sending bot logs message", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
