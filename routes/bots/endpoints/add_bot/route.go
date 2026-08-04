// Package add_bot implements PUT /bots — "Add Bot".
//
// Adds a bot to the database. Returns 204 on success
package add_bot

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"regexp"
	"slices"
	"strings"
	"time"

	"popplio/api"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
)

func createBotsArgs(bot types.CreateBot) []any {
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
		bot.TeamOwner,
		bot.GuildCount,
		bot.VanityRef,
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
		return resp.Err("Error calculating ratelimits", err, zap.String("userID", d.Auth.ID))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
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
		return resp.BadRequest(err.Error())
	}

	// Check if the bot is already in the database
	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", payload.BotID).Scan(&count)

	if err != nil {
		return resp.Err("Error while checking if bot is already in database", err, zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
	}

	if count > 0 {
		return resp.Conflict("This bot is already in the database")
	}

	// Ensure the bot actually exists right now
	bot, err := dovewing.GetUser(d.Context, payload.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.BadRequest("This bot does not exist: " + err.Error())
	}

	if !bot.Bot {
		return resp.BadRequest("This user is not a bot")
	}

	metadata, err := assets.CheckBot(d.Context, payload.BotID, payload.ClientID)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	if metadata.BotID != payload.BotID {
		return resp.BadRequest("The bot ID provided does not match the bot ID found")
	}

	if metadata.ListType != "" {
		return resp.BadRequest("This bot is already in the database")
	}

	if !metadata.BotPublic {
		return resp.BadRequest("Bot is not public")
	}

	// Set guild count from metadata
	payload.GuildCount = &metadata.GuildCount

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
		return resp.Err("Error while checking if calculated vanity is already taken", err, zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
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

	// Save the bot to the database
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
	}

	defer tx.Rollback(d.Context)

	// Setup teams
	if d.Auth.TargetType == api.TargetTypeTeam {
		payload.TeamOwner = d.Auth.ID
	}

	// Check team owner here, to avoid a race condition
	if payload.TeamOwner != "" {
		entityPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", payload.TeamOwner)

		if err != nil {
			state.Logger.Error("Error while getting team perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("teamID", payload.TeamOwner), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
			return resp.BadRequest("Error getting user perms: " + err.Error())
		}

		if !entityPerms.Has(perms.EntityAddBots) {
			return resp.Forbidden("You do not have permission to add new bots to this team")
		}
	} else {
		// Create new team
		var teamId = uuid.New()

		var vanityRef string
		err = tx.QueryRow(d.Context, "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'team', $2) RETURNING itag", teamId, metadata.Name+crypto.RandString(16)).Scan(&vanityRef)

		if err != nil {
			return resp.BadRequest("Error while creating vanity: " + err.Error())
		}

		_, err = tx.Exec(d.Context, "INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'api/add_bot')", teamId, metadata.Name, vanityRef)

		if err != nil {
			return resp.BadRequest("Error while creating vanity: " + err.Error())
		}

		// Add the team member to the team as well
		_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'api/add_bot')", teamId, d.Auth.ID, []string{"global.*"})

		if err != nil {
			return resp.BadRequest("Error while adding team member: " + err.Error())
		}

		payload.TeamOwner = teamId.String()
	}

	// Create vanity
	var itag pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, $3) RETURNING itag", vanity, payload.BotID, "bot").Scan(&itag)

	if err != nil {
		return resp.Err("Error while inserting vanity", err, zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID), zap.String("vanity", vanity))
	}

	// Lastly, set the vanity ref correctly
	payload.VanityRef = itag

	// Get the arguments to pass when adding the bot
	botArgs := createBotsArgs(payload)

	if len(createBotsColsArr) != len(botArgs) {
		return resp.ErrBody("createBotsColsArr and botArgs do not match in length", "Internal Error: The number of columns and arguments do not match", nil, zap.Any("createBotsColsArr", createBotsColsArr), zap.Any("botArgs", botArgs))
	}

	_, err = tx.Exec(d.Context, "INSERT INTO bots ("+createBotsCols+") VALUES ("+createBotsParams+")", botArgs...)

	if err != nil {
		return resp.Err("Error while inserting bot", err, zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
	}

	botAddedEmbed := discord.Embed{
		URL:   state.Config.Sites.Frontend.Parse() + "/bots/" + payload.BotID,
		Title: "New Bot Added",
		Fields: []discord.EmbedField{
			{
				Name:   "Name",
				Value:  metadata.Name,
				Inline: validators.TruePtr,
			},
			{
				Name:   "Bot ID",
				Value:  payload.BotID,
				Inline: validators.TruePtr,
			},
			{
				Name: "Owner",
				Value: func() string {
					if payload.TeamOwner != "" {
						return fmt.Sprintf("[Team %s](%s/teams/%s)", payload.TeamOwner, state.Config.Sites.Frontend.Parse(), payload.TeamOwner)
					}
					return fmt.Sprintf("<@%s>", d.Auth.ID)
				}(),
				Inline: validators.TruePtr,
			},
		},
	}

	// An EmbedResource with an empty URL is itself invalid and gets the
	// whole message rejected (50035) — Discord wants the field omitted
	// entirely, not present-but-empty, when there's no avatar.
	if metadata.Avatar != "" {
		botAddedEmbed.Thumbnail = &discord.EmbedResource{URL: metadata.Avatar}
	}

	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
		Content: state.Config.Meta.UrgentMentions,
		Embeds:  []discord.Embed{botAddedEmbed},
	})

	if err != nil {
		state.Logger.Error("Error while sending bot logs message", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", payload.BotID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
