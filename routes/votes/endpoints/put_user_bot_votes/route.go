package put_user_bot_votes

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/bothooks_legacy"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Bot Vote",
		Description: "Creates a vote for a bot. **For internal use only**. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
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

	id, err := utils.ResolveBot(d.Context, chi.URLParam(r, "bid"))

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	userId := chi.URLParam(r, "uid")

	var voteBannedState bool

	err = state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", userId).Scan(&voteBannedState)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBannedState {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You are banned from voting right now! Contact support if you think this is a mistake",
				Error:   true,
			},
		}
	}

	var botType pgtype.Text
	var voteBannedBotsState bool

	err = state.Pool.QueryRow(d.Context, "SELECT type, vote_banned FROM bots WHERE bot_id = $1", id).Scan(&botType, &voteBannedBotsState)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBannedBotsState {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "This bot is banned from being voted on right now! Contact support if you think this is a mistake",
				Error:   true,
			},
		}
	}

	if botType.String != "approved" && botType.String != "certified" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Woah there, this bot needs to be approved before you can vote for it!",
				Error:   true,
			},
		}
	}

	voteParsed, err := utils.GetVoteData(d.Context, userId, id, true)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if voteParsed.HasVoted {
		timeElapsed := time.Now().UnixMilli() - voteParsed.LastVoteTime
		state.Logger.Info(timeElapsed)

		timeToWait := int64(voteParsed.VoteInfo.VoteTime)*60*60*1000 - timeElapsed

		timeToWaitTime := (time.Duration(timeToWait) * time.Millisecond)

		hours := timeToWaitTime / time.Hour
		mins := (timeToWaitTime - (hours * time.Hour)) / time.Minute
		secs := (timeToWaitTime - (hours*time.Hour + mins*time.Minute)) / time.Second

		timeStr := fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", hours, mins, secs)

		var alreadyVotedMsg = types.ApiError{
			Message: "Please wait " + timeStr + " before voting again",
			Error:   true,
		}

		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   alreadyVotedMsg,
		}
	}

	// Record new vote
	var itag pgtype.UUID
	err = state.Pool.QueryRow(d.Context, "INSERT INTO votes (user_id, bot_id) VALUES ($1, $2) RETURNING itag", userId, id).Scan(&itag)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var oldVotes pgtype.Int4

	err = state.Pool.QueryRow(d.Context, "SELECT votes FROM bots WHERE bot_id = $1", id).Scan(&oldVotes)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var incr = 1
	var votes = oldVotes.Int32

	if utils.GetDoubleVote() {
		incr = 2
		votes += 2
	} else {
		votes++
	}

	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET votes = votes + $1 WHERE bot_id = $2", incr, id)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	userObj, err := dovewing.GetDiscordUser(d.Context, userId)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	botObj, err := dovewing.GetDiscordUser(d.Context, id)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.VoteLogs, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + id,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: botObj.Avatar,
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart:" + userObj.DisplayName + " has voted for " + botObj.Username,
				Color:       0x8A6BFD,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Vote Count:",
						Value:  strconv.Itoa(int(votes)),
						Inline: true,
					},
					{
						Name:   "User ID:",
						Value:  userObj.ID,
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[View " + botObj.Username + "](https://botlist.site/" + id + ")",
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[Vote for " + botObj.Username + "](https://botlist.site/" + id + "/vote)",
						Inline: true,
					},
				},
			},
		},
	})

	if err != nil {
		state.Logger.Warn(err)
	}

	// Send webhook in a goroutine refunding the vote if it failed
	go func() {
		var webhooksV2 bool

		err := state.Pool.QueryRow(state.Context, "SELECT webhooks_v2 FROM bots WHERE bot_id = $1", id).Scan(&webhooksV2)

		if err != nil {
			state.Logger.Error(err)
			return
		}

		if webhooksV2 {
			state.Logger.Info("Sending webhook for vote (v2) for " + id)

			err := bothooks.Send(bothooks.With[events.WebhookBotVoteData]{
				UserID: userId,
				BotID:  id,
				Data: events.WebhookBotVoteData{
					Votes: int(votes),
					Test:  false,
				},
			})

			if err != nil {
				state.Logger.Error(err)
				return
			}

			return
		}

		err = bothooks_legacy.SendLegacy(bothooks_legacy.WebhookPostLegacy{
			BotID:  id,
			UserID: userId,
			Votes:  int(votes),
		})

		var msg types.Alert

		if err != nil && err.Error() == "httpUser" {
			msg = types.Alert{
				Type:    types.AlertTypeWarning,
				Title:   "Vote Rewards Deferred!",
				Message: botObj.Username + " uses the HTTP API for votes. Vote rewards may take time to register.",
				Icon:    botObj.Avatar,
				URL: pgtype.Text{
					String: "https://botlist.site/" + id + "/vote",
					Valid:  true,
				},
			}
		} else if err != nil {
			msg = types.Alert{
				Type:    types.AlertTypeError,
				Title:   "Whoa There!",
				Message: "We couldn't notify " + botObj.Username + ": " + err.Error() + ".",
				Icon:    botObj.Avatar,
				URL: pgtype.Text{
					String: "https://botlist.site/" + id + "/vote",
					Valid:  true,
				},
			}
		} else {
			msg = types.Alert{
				Type:    types.AlertTypeSuccess,
				Title:   "Bot Notified!",
				Message: "Successfully alerted " + botObj.Username + " to your vote with ID of " + id + ".",
				Icon:    botObj.Avatar,
				URL: pgtype.Text{
					String: "https://botlist.site/" + id + "/vote",
					Valid:  true,
				},
			}
		}

		err = notifications.PushNotification(userId, msg)

		if err != nil {
			state.Logger.Error(err)
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
