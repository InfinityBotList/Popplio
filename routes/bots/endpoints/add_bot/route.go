package add_bot

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"popplio/api"
	"popplio/ratelimit"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"
	"github.com/infinitybotlist/eureka/crypto"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
)

type internalData struct {
	QueueName   *string
	QueueAvatar *string
	Owner       string
	Vanity      *string
	GuildCount  *int
}

type CreateBot struct {
	BotID      string       `db:"bot_id" json:"bot_id" validate:"required,numeric" msg:"Bot ID must be numeric"`                                       // impld
	ClientID   string       `db:"client_id" json:"client_id" validate:"required,numeric" msg:"Client ID must be numeric"`                              // impld
	Short      string       `db:"short" json:"short" validate:"required,min=30,max=150" msg:"Short description must be between 30 and 150 characters"` // impld
	Long       string       `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                 // impld
	Prefix     string       `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`              // impld
	Invite     string       `db:"invite" json:"invite" validate:"required,https" msg:"Invite is required and must be a valid HTTPS URL"`               // impld
	Banner     *string      `db:"banner" json:"banner" validate:"omitempty,https" msg:"Background must be a valid HTTPS URL"`                          // impld
	Library    string       `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`           // impld
	ExtraLinks []types.Link `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                   // Impld
	Tags       []string     `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	NSFW       bool         `db:"nsfw" json:"nsfw"`
	StaffNote  *string      `db:"approval_note" json:"staff_note" validate:"omitempty,max=512" msg:"Staff note must be less than 512 characters if sent"` // impld

	// Not needed to send
	QueueName   *string `db:"queue_name" json:"-"`
	QueueAvatar *string `db:"queue_avatar" json:"-"`
	Owner       string  `db:"owner" json:"-"`
	Vanity      *string `db:"vanity" json:"-"`
	GuildCount  *int    `db:"servers" json:"-"`
}

func createBotsArgs(bot CreateBot, id internalData) []any {
	return []any{
		bot.BotID,
		bot.ClientID,
		bot.Short,
		bot.Long,
		bot.Prefix,
		bot.Invite,
		bot.Banner,
		bot.Library,
		bot.ExtraLinks,
		bot.Tags,
		bot.NSFW,
		bot.StaffNote,
		id.QueueName,
		id.QueueAvatar,
		id.Owner,
		id.Vanity,
		id.GuildCount,
	}
}

var (
	compiledMessages = api.CompileValidationErrors(CreateBot{})

	createBotsColsArr = utils.GetCols(CreateBot{})
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
		Summary:     "Create Bot",
		Description: "Adds a bot to the database. The main owner will be the user who created the bot. Returns 204 on success",
		Req:         CreateBot{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	if d.Auth.ID != "728871946456137770" {
		return api.HttpResponse{
			Status: http.StatusNotImplemented,
			Json: types.ApiError{
				Message: "This endpoint is temporarily under maintenance for some important fixups",
				Error:   true,
			},
		}
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "add_bot",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var payload CreateBot

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	err = utils.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: err.Error(),
				Error:   true,
			},
		}
	}

	// Check if the bot is already in the database
	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", payload.BotID).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count > 0 {
		return api.HttpResponse{
			Status: http.StatusConflict,
			Json: types.ApiError{
				Message: "This bot is already in the database",
				Error:   true,
			},
		}
	}

	// Ensure the bot actually exists right now
	bot, err := dovewing.GetDiscordUser(d.Context, payload.BotID)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This bot does not exist: " + err.Error(),
				Error:   true,
			},
		}
	}

	if !bot.Bot {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This user is not a bot",
				Error:   true,
			},
		}
	}

	// Ensure the main owner exists
	_, err = dovewing.GetDiscordUser(d.Context, d.Auth.ID)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "The main owner of this bot somehow does not exist: " + err.Error(),
				Error:   true,
			},
		}
	}

	metadata, err := assets.CheckBot(payload.BotID, payload.ClientID)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: err.Error(),
				Error:   true,
			},
		}
	}

	if metadata.BotID != payload.BotID {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "The bot ID provided does not match the bot ID found",
				Error:   true,
			},
		}
	}

	if metadata.ListType != "" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This bot is already in the database",
				Error:   true,
			},
		}
	}

	if !metadata.BotPublic {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Bot is not public",
				Error:   true,
			},
		}
	}

	id := internalData{}

	id.QueueName = &metadata.Name
	id.QueueAvatar = &metadata.Avatar
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

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE lower(vanity) = $1", vanity).Scan(&vanityCount)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if vanityCount > 0 {
		newVanity := vanity + "-" + crypto.RandString(5)
		id.Vanity = &newVanity
	} else {
		id.Vanity = &vanity
	}

	// Get the arguments to pass when adding the bot
	botArgs := createBotsArgs(payload, id)

	if len(createBotsColsArr) != len(botArgs) {
		state.Logger.Error(botArgs, createBotsColsArr)
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal Error: The number of columns and arguments do not match",
				Error:   true,
			},
		}
	}

	// Save the bot to the database
	_, err = state.Pool.Exec(d.Context, "INSERT INTO bots ("+createBotsCols+") VALUES ("+createBotsParams+")", botArgs...)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	utils.ClearUserCache(d.Context, d.Auth.ID)

	state.Discord.ChannelMessageSendComplex(state.Config.Channels.BotLogs, &discordgo.MessageSend{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []*discordgo.MessageEmbed{
			{
				URL:   state.Config.Sites.Frontend + "/bots/" + payload.BotID,
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
						Name:   "Main Owner",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: true,
					},
				},
			},
		},
	})

	return api.DefaultResponse(http.StatusNoContent)
}
