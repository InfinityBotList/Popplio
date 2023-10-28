package put_user_entity_votes

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"popplio/config"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/bothooks_legacy"
	"popplio/webhooks/events"
	"popplio/webhooks/serverhooks"
	"popplio/webhooks/teamhooks"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Entity Vote",
		Description: "Creates a vote for an entity. Returns 204 on success. Note that for compatibility, a trailing 's' is removed",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
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
				Name:        "target_id",
				Description: "The bot ID",
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
		state.Logger.Error("Failed to unmarshal hcaptcha response", zap.Error(err))
	} else {
		// We have a response, lets verify it
		resp, err := http.PostForm("https://hcaptcha.com/siteverify", url.Values{
			"secret":   {state.Config.Hcaptcha.Secret},
			"response": {hcaptchaResp.Response},
		})

		if err != nil {
			state.Logger.Error("Failed to verify hcaptcha", zap.Error(err))
			return
		}

		defer resp.Body.Close()

		var hcaptchaResp struct {
			Success    bool     `json:"success"`
			ErrorCodes []string `json:"error-codes"`
		}

		err = json.NewDecoder(resp.Body).Decode(&hcaptchaResp)

		if err != nil {
			state.Logger.Error("Failed to decode hcaptcha response", zap.Error(err))
			return
		}

		if !hcaptchaResp.Success {
			state.Logger.Error("hcaptcha failed to verify token", zap.Strings("errorCodes", hcaptchaResp.ErrorCodes))
			return
		}

		state.Logger.Info("hcaptcha siteverify check passed")
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

	targetType = strings.TrimSuffix(targetType, "s")

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
		state.Logger.Error("Failed to check if user is vote banned", zap.Error(err), zap.String("userId", uid))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBanned {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are banned from voting right now! Contact support if you think this is a mistake"},
		}
	}

	// Handle entity specific checks here, such as ensuring the entity actually exists
	switch targetType {
	case "bot":
		if upvote == "false" {
			return uapi.HttpResponse{
				Status: http.StatusNotImplemented,
				Json:   types.ApiError{Message: "Downvoting bots is not implemented yet"},
			}
		}
	}

	entityInfo, err := votes.GetEntityInfo(d.Context, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to fetch entity info", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error: " + err.Error()},
		}
	}

	// Now check the vote
	vi, err := votes.EntityVoteCheck(d.Context, uid, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to check vote", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
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

	if err != nil {
		state.Logger.Error("Failed to create transaction [put_user_entity_votes]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Keep adding votes until, but not including vi.VoteInfo.PerUser
	for i := 0; i < vi.VoteInfo.PerUser; i++ {
		_, err = tx.Exec(d.Context, "INSERT INTO entity_votes (author, target_id, target_type, upvote, vote_num) VALUES ($1, $2, $3, $4, $5)", uid, targetId, targetType, upvote == "true", i)

		if err != nil {
			state.Logger.Error("Failed to insert vote", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType), zap.String("upvote", upvote))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Fetch new vote count
	nvc, err := votes.EntityGetVoteCount(d.Context, tx, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to fetch new vote count", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Entity specific handling here, if desired
	//
	// Note that `votes` is a cached value based on new vote count
	switch targetType {
	case "bot":
		_, err = tx.Exec(d.Context, "UPDATE bots SET votes = $1 WHERE bot_id = $2", nvc, targetId)

		if err != nil {
			state.Logger.Error("Failed to update bot vote count cache", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "pack":
		_, err = tx.Exec(d.Context, "UPDATE packs SET votes = $1 WHERE url = $2", nvc, targetId)

		if err != nil {
			state.Logger.Error("Failed to update pack vote count cache", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "team":
		_, err = tx.Exec(d.Context, "UPDATE teams SET votes = $1 WHERE id = $2", nvc, targetId)

		if err != nil {
			state.Logger.Error("Failed to update team vote count cache", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "server":
		_, err = tx.Exec(d.Context, "UPDATE servers SET votes = $1 WHERE server_id = $2", nvc, targetId)

		if err != nil {
			state.Logger.Error("Failed to update server vote count cache", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Commit transaction
	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Failed to commit transaction", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Fetch user info to log it to server
	userObj, err := dovewing.GetUser(d.Context, uid, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to fetch user info", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

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
						Name:   "Votes Added:",
						Value:  strconv.Itoa(vi.VoteInfo.PerUser),
						Inline: true,
					},
					{
						Name:   "User ID:",
						Value:  userObj.ID,
						Inline: true,
					},
					{
						Name:   "View " + targetType + "'s page",
						Value:  "[View " + entityInfo.Name + "](" + entityInfo.URL + ")",
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[Vote for " + entityInfo.Name + "](" + entityInfo.VoteURL + ")",
						Inline: true,
					},
				},
			},
		},
	})

	if err != nil {
		state.Logger.Error("Failed to send vote log message", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	// Send webhook in a goroutine refunding the vote if it failed
	go func() {
		err = nil // Be sure error is empty before we start

		if targetType == "bot" && config.UseLegacyWebhooks(targetId) {
			state.Logger.Info("Using legacy webhooks", zap.String("targetId", targetId), zap.String("targetType", targetType), zap.String("userId", uid))

			err = bothooks_legacy.SendLegacy(bothooks_legacy.WebhookPostLegacy{
				BotID:  targetId,
				UserID: uid,
				Votes:  nvc,
			})

			if err != nil {
				state.Logger.Error("Failed to send legacy webhook", zap.Error(err))
			}

			return
		}

		switch targetType {
		case "bot":
			err = bothooks.Send(bothooks.With{
				UserID: uid,
				BotID:  targetId,
				Data: events.WebhookBotVoteData{
					Votes:   nvc,
					PerUser: vi.VoteInfo.PerUser,
				},
			})
		case "team":
			err = teamhooks.Send(teamhooks.With{
				UserID: uid,
				TeamID: targetId,
				Data: events.WebhookTeamVoteData{
					Votes:    nvc,
					PerUser:  vi.VoteInfo.PerUser,
					Downvote: upvote == "false",
				},
			})
		case "server":
			err = serverhooks.Send(serverhooks.With{
				UserID:   uid,
				ServerID: targetId,
				Data: events.WebhookServerVoteData{
					Votes:    nvc,
					PerUser:  vi.VoteInfo.PerUser,
					Downvote: upvote == "false",
				},
			})
		}

		var msg types.Alert

		if err != nil {
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
			if entityInfo != nil {
				msg = types.Alert{
					Type:    types.AlertTypeSuccess,
					Title:   cases.Title(language.English).String(targetType) + " Notified!",
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
					Title:   cases.Title(language.English).String(targetType) + " Notified!",
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
			state.Logger.Error("Failed to send push notification", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
