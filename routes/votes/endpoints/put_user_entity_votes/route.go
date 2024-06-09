package put_user_entity_votes

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/events"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
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
				Description: "The target ID",
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

	// Check if upvote query parameter is valid
	upvoteStr := r.URL.Query().Get("upvote")

	if upvoteStr != "true" && upvoteStr != "false" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "upvote must be either `true` or `false`"},
		}
	}

	upvote := upvoteStr == "true"

	// Check if user is allowed to even make a vote right now.
	var voteBanned bool

	err = state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", uid).Scan(&voteBanned)

	if err != nil {
		state.Logger.Error("Failed to check if user is vote banned", zap.Error(err), zap.String("userId", uid))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error checking if user is vote banned: " + err.Error()},
		}
	}

	if voteBanned {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are banned from voting right now! Contact support if you think this is a mistake"},
		}
	}

	entityInfo, err := votes.GetEntityInfo(d.Context, state.Pool, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to fetch entity info", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error: " + err.Error()},
		}
	}

	// Now check the vote
	vi, err := votes.EntityVoteCheck(d.Context, state.Pool, uid, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to check vote", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !vi.VoteInfo.SupportsDownvotes && !upvote {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "This entity does not support downvotes"},
		}
	}

	if !vi.VoteInfo.SupportsUpvotes && upvote {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "This entity does not support upvotes"},
		}
	}

	if vi.HasVoted {
		// If !Multiple Votes
		if !vi.VoteInfo.MultipleVotes {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "You have already voted for this entity before!"},
			}
		}

		var timeStr string
		if vi.Wait != nil {
			timeStr = fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", vi.Wait.Hours, vi.Wait.Minutes, vi.Wait.Seconds)
		} else {
			timeStr = "a while"
		}

		if len(vi.ValidVotes) > 1 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Your last vote was a double vote, calm down for " + timeStr + "?"},
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
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Failed to create transaction: " + err.Error()},
		}
	}

	defer tx.Rollback(d.Context)

	// Keep adding votes until, but not including vi.VoteInfo.PerUser
	for i := 0; i < vi.VoteInfo.PerUser; i++ {
		_, err = tx.Exec(d.Context, "INSERT INTO entity_votes (author, target_id, target_type, upvote, vote_num) VALUES ($1, $2, $3, $4, $5)", uid, targetId, targetType, upvote, i)

		if err != nil {
			state.Logger.Error("Failed to insert vote", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType), zap.Bool("upvote", upvote))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	// Fetch new vote count
	nvc, err := votes.EntityGetVoteCount(d.Context, tx, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to fetch new vote count", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Commit transaction
	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Failed to commit transaction", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Fetch user info to log it to server
	go func() {
		userObj, err := dovewing.GetUser(d.Context, uid, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch user info", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return
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
	}()

	// Send webhook in a goroutine
	go func() {
		err = nil // Be sure error is empty before we start

		err = drivers.Send(drivers.With{
			UserID:     uid,
			TargetID:   targetId,
			TargetType: targetType,
			Data: events.WebhookNewVoteData{
				Votes:   nvc,
				PerUser: vi.VoteInfo.PerUser,
			},
		})

		if err != nil {
			state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
