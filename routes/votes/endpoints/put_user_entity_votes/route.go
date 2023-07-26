package put_user_entity_votes

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"popplio/config"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/votes"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/bothooks_legacy"
	"popplio/webhooks/events"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Internal struct that stores internal entity info
type entityIdent struct {
	Name    string
	URL     string
	VoteURL string
	Avatar  string
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Entity Vote",
		Description: "Creates a vote for an entity. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "upvote",
				Description: "Whether or not to upvote the entity. Must be either true or false",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func hcaptcha(b []byte) {
	// OK, so we can handle hcaptcha
	state.Logger.Info("Trying to handle hcaptcha")
	var hcaptchaResp struct {
		Key      string `json:"key"`
		Response string `json:"response"`
	}

	err := json.Unmarshal(b, &hcaptchaResp)

	if err != nil {
		state.Logger.Error(err)
	} else {
		// We have a response, lets verify it
		resp, err := http.PostForm("https://hcaptcha.com/siteverify", url.Values{
			"secret":   {state.Config.Hcaptcha.Secret},
			"response": {hcaptchaResp.Response},
		})

		if err != nil {
			state.Logger.Error(err)
			return
		}

		defer resp.Body.Close()

		var hcaptchaResp struct {
			Success    bool     `json:"success"`
			ErrorCodes []string `json:"error-codes"`
		}

		err = json.NewDecoder(resp.Body).Decode(&hcaptchaResp)

		if err != nil {
			state.Logger.Error(err)
			return
		}

		if !hcaptchaResp.Success {
			state.Logger.Error("hcaptcha failed" + fmt.Sprintf("%v", hcaptchaResp.ErrorCodes))
			return
		}

		state.Logger.Info("hcaptcha passed")
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Try reading body if its there to handle hcaptcha
	bytes, err := io.ReadAll(r.Body)

	if err == nil && len(bytes) > 0 {
		go hcaptcha(bytes)
	}

	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := chi.URLParam(r, "target_type")

	if uid == "" || targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	err = utils.StagingCheckSensitive(d.Context, uid)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	upvote := r.URL.Query().Get("upvote")

	if upvote != "true" && upvote != "false" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "upvote must be either true or false"},
		}
	}

	// Check if user is allowed to even make a vote right now.
	var voteBanned bool

	err = state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", uid).Scan(&voteBanned)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBanned {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are banned from voting right now! Contact support if you think this is a mistake"},
		}
	}

	var entityInfo *entityIdent

	// Handle entity specific checks here, such as ensuring the entity actually exists
	switch targetType {
	case "bot":
		if upvote == "false" {
			return uapi.HttpResponse{
				Status: http.StatusNotImplemented,
				Json:   types.ApiError{Message: "Downvoting bots is not implemented yet"},
			}
		}

		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", targetId).Scan(&count)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.DefaultResponse(http.StatusNotFound)
		}

		var botType string
		var voteBanned bool

		err = state.Pool.QueryRow(d.Context, "SELECT type, vote_banned FROM bots WHERE bot_id = $1", targetId).Scan(&botType, &voteBanned)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if voteBanned {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "This bot is banned from being voted on right now! Contact support if you think this is a mistake"},
			}
		}

		if botType != "approved" && botType != "certified" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Woah there, this bot needs to be approved before you can vote for it!"},
			}
		}

		botObj, err := dovewing.GetUser(d.Context, targetId, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		// Set entityInfo for log
		entityInfo = &entityIdent{
			URL:     "https://botlist.site/" + targetId,
			VoteURL: "https://botlist.site/" + targetId + "/vote",
			Name:    botObj.Username,
			Avatar:  botObj.Avatar,
		}
	case "pack":
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.DefaultResponse(http.StatusNotFound)
		}

	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "Support for this target type has not been implemented yet"},
		}
	}

	// Now check the vote
	vi, err := votes.EntityVoteCheck(d.Context, uid, targetId, targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if vi.HasVoted {
		timeStr := fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", vi.Wait.Hours, vi.Wait.Minutes, vi.Wait.Seconds)

		if len(vi.ValidVotes) > 1 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Your last vote was a double vote, calm down?: " + timeStr},
			}
		}

		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Please wait " + timeStr + " before voting again"},
		}
	}

	// Create a new entity vote
	tx, err := state.Pool.Begin(d.Context)
	defer tx.Rollback(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = tx.Exec(d.Context, "INSERT INTO entity_votes (author, target_id, target_type, upvote) VALUES ($1, $2, $3, $4)", uid, targetId, targetType, upvote == "true")

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if vi.VoteInfo.DoubleVotes {
		// Create a second vote
		_, err = tx.Exec(d.Context, "INSERT INTO entity_votes (author, target_id, target_type, upvote) VALUES ($1, $2, $3, $4)", uid, targetId, targetType, upvote == "true")

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Fetch new vote count
	nvc, err := votes.EntityGetVoteCount(d.Context, tx, targetId, targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Entity specific handling here, if desired
	//
	// Note that `votes` is a cached value based on new vote count
	switch targetType {
	case "bot":
		_, err = tx.Exec(d.Context, "UPDATE bots SET votes = $1 WHERE bot_id = $2", nvc, targetId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "pack":
		_, err = tx.Exec(d.Context, "UPDATE packs SET votes = $1 WHERE url = $2", nvc, targetId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Commit transaction
	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Fetch user info to log it to server
	userObj, err := dovewing.GetUser(d.Context, uid, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if entityInfo != nil {
		_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.VoteLogs, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL: entityInfo.URL,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: entityInfo.Avatar,
					},
					Title:       "🎉 Vote Count Updated!",
					Description: ":heart:" + userObj.DisplayName + " has voted for " + targetType + ": " + entityInfo.Name,
					Color:       0x8A6BFD,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Vote Count:",
							Value:  strconv.Itoa(nvc),
							Inline: true,
						},
						{
							Name:   "User ID:",
							Value:  userObj.ID,
							Inline: true,
						},
						{
							Name:   "Vote Page",
							Value:  "[View " + entityInfo.Name + "](" + entityInfo.URL + ")",
							Inline: true,
						},
						{
							Name:   "Vote Page",
							Value:  "[Vote for " + entityInfo.Name + entityInfo.VoteURL,
							Inline: true,
						},
					},
				},
			},
		})

		if err != nil {
			state.Logger.Warn(err)
		}
	}

	// Send webhook in a goroutine refunding the vote if it failed
	go func() {
		err = nil // Be sure error is empty before we start

		if targetType == "bot" && config.UseLegacyWebhooks(targetId) {
			state.Logger.Info("Using legacy webhooks", zap.String("targetId", targetId), zap.String("targetType", targetType), zap.String("uid", uid))

			err = bothooks_legacy.SendLegacy(bothooks_legacy.WebhookPostLegacy{
				BotID:  targetId,
				UserID: uid,
				Votes:  nvc,
			})
		} else {
			switch targetType {
			case "bot":
				err = bothooks.Send(bothooks.With[events.WebhookBotVoteData]{
					UserID: uid,
					BotID:  targetId,
					Data: events.WebhookBotVoteData{
						Votes: nvc,
					},
				})
			}
		}

		var msg types.Alert

		if err != nil {
			// Check if the entity follows the entityInfo protocol, if not, fallback
			if entityInfo != nil {
				msg = types.Alert{
					Type:    types.AlertTypeError,
					Title:   "Whoa There!",
					Message: "We couldn't notify " + targetType + " " + entityInfo.Name + ": " + err.Error() + ".",
					Icon:    entityInfo.Avatar,
					URL: pgtype.Text{
						String: entityInfo.VoteURL,
						Valid:  true,
					},
				}
			} else {
				msg = types.Alert{
					Type:    types.AlertTypeError,
					Title:   "Whoa There!",
					Message: "We couldn't notify " + targetType + " " + targetId + ": " + err.Error() + ".",
					URL: pgtype.Text{
						String: "https://botlist.site/" + targetType + "/" + targetId,
						Valid:  true,
					},
				}
			}
		} else {
			// Check if the entity follows the entityInfo protocol, if not, fallback
			if entityInfo != nil {
				msg = types.Alert{
					Type:    types.AlertTypeSuccess,
					Title:   "Bot Notified!",
					Message: "Successfully alerted " + targetType + " " + entityInfo.Name + " to your vote with target ID of " + targetId + ".",
					Icon:    entityInfo.Avatar,
					URL: pgtype.Text{
						String: entityInfo.VoteURL,
						Valid:  true,
					},
				}
			} else {
				msg = types.Alert{
					Type:    types.AlertTypeSuccess,
					Title:   "Bot Notified!",
					Message: "Successfully alerted " + targetType + " " + targetId + " to your vote with target ID of " + targetId + ".",
					URL: pgtype.Text{
						String: "https://botlist.site/" + targetType + "/" + targetId,
						Valid:  true,
					},
				}

			}
		}

		err = notifications.PushNotification(uid, msg)

		if err != nil {
			state.Logger.Error(err)
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
