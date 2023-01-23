package put_user_bot_votes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"

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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id, err := utils.ResolveBot(d.Context, chi.URLParam(r, "bid"))

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	userId := chi.URLParam(r, "uid")

	var voteBannedState bool

	err = state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", userId).Scan(&voteBannedState)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBannedState {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Data:   constants.VoteBanned,
		}
	}

	var botType pgtype.Text
	var voteBannedBotsState bool

	err = state.Pool.QueryRow(d.Context, "SELECT type, vote_banned FROM bots WHERE bot_id = $1", id).Scan(&botType, &voteBannedBotsState)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if voteBannedBotsState {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Data:   constants.VoteBanned,
		}
	}

	if botType.String != "approved" && botType.String != "certified" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   constants.NotApproved,
		}
	}

	voteParsed, err := utils.GetVoteData(d.Context, userId, id, true)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if voteParsed.HasVoted {
		timeElapsed := time.Now().UnixMilli() - voteParsed.LastVoteTime
		state.Logger.Info(timeElapsed)

		timeToWait := int64(utils.GetVoteTime())*60*60*1000 - timeElapsed

		timeToWaitTime := (time.Duration(timeToWait) * time.Millisecond)

		hours := timeToWaitTime / time.Hour
		mins := (timeToWaitTime - (hours * time.Hour)) / time.Minute
		secs := (timeToWaitTime - (hours*time.Hour + mins*time.Minute)) / time.Second

		timeStr := fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", hours, mins, secs)

		var alreadyVotedMsg = types.ApiError{
			Message: "Please wait " + timeStr + " before voting again",
			Error:   true,
		}

		return api.HttpResponse{
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
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var oldVotes pgtype.Int4

	err = state.Pool.QueryRow(d.Context, "SELECT votes FROM bots WHERE bot_id = $1", id).Scan(&oldVotes)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
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
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	userObj, err := utils.GetDiscordUser(userId)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	botObj, err := utils.GetDiscordUser(id)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.VoteLogs, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + id,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: botObj.Avatar,
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart:" + userObj.Username + "#" + userObj.Discriminator + " has voted for " + botObj.Username,
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
		err = webhooks.Send(types.WebhookPost{
			BotID:  id,
			UserID: userId,
			Votes:  int(votes),
		})

		var msg notifications.Message

		if err.Error() == "httpUser" {
			state.Pool.Exec(
				state.Context,
				"INSERT INTO alerts (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
				userId,
				"https://infinitybots.gg/bots/"+id,
				"This bot uses the Get All Bot Votes HTTP API to handle vote rewards. ID: "+id+" ("+botObj.Username+")",
				"info",
			)
			msg = notifications.Message{
				Title:   "Vote Rewards Deferred!",
				Message: botObj.Username + " uses the HTTP API for votes. Vote rewards may take time to register.",
			}
		} else if err != nil {
			state.Pool.Exec(
				state.Context,
				"INSERT INTO alerts (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
				userId,
				"https://infinitybots.gg/bots/"+id,
				"Something went wrong when notifying this bot. The error was: "+err.Error()+".",
				"error",
			)
			msg = notifications.Message{
				Title:   "Whoa There!",
				Message: "We couldn't notify " + botObj.Username + ": " + err.Error() + ".",
				Icon:    botObj.Avatar,
			}
		} else {
			state.Pool.Exec(
				state.Context,
				"INSERT INTO alerts (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
				userId,
				"https://infinitybots.gg/bots/"+id,
				"Successfully alerted this bot to your vote with ID of "+id+"("+botObj.Username+")",
				"info",
			)
			msg = notifications.Message{
				Title:   "Bot Notified!",
				Message: "Successfully alerted " + botObj.Username + " to your vote with ID of " + id + ".",
			}
		}

		notifIds, err := state.Pool.Query(state.Context, "SELECT notif_id FROM poppypaw WHERE user_id = $1", userId)

		if err != nil {
			state.Logger.Error(err)
			return
		}

		defer notifIds.Close()

		for notifIds.Next() {
			var notifId string

			err = notifIds.Scan(&notifId)

			if err != nil {
				state.Logger.Error(err)
				continue
			}

			bytes, err := json.Marshal(msg)

			if err != nil {
				state.Logger.Error(err)
				continue
			}

			notifications.NotifChannel <- notifications.Notification{
				NotifID: notifId,
				Message: bytes,
			}
		}
	}()

	return api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
