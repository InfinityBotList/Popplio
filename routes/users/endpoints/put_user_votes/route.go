package put_user_votes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() {
	docs.Route(&docs.Doc{
		Method:      "PUT",
		Path:        "/users/{uid}/bots/{bid}/votes",
		OpId:        "put_user_votes",
		Summary:     "Create User Vote",
		Description: "Creates a vote for a bot. **For internal use only**",
		Tags:        []string{api.CurrentTag},
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
		Resp:     types.ApiError{},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var vars = map[string]string{
		"uid": chi.URLParam(r, "uid"),
		"bid": chi.URLParam(r, "bid"),
	}

	var botId pgtype.Text
	var botType pgtype.Text

	var voteBannedState bool

	err := state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", vars["uid"]).Scan(&voteBannedState)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	if voteBannedState {
		d.Resp <- types.HttpResponse{
			Status: http.StatusForbidden,
			Data:   constants.VoteBanned,
		}
		return
	}

	var voteBannedBotsState bool

	err = state.Pool.QueryRow(d.Context, "SELECT bot_id, type, vote_banned FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId, &botType, &voteBannedBotsState)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	if voteBannedBotsState {
		d.Resp <- types.HttpResponse{
			Status: http.StatusForbidden,
			Data:   constants.VoteBanned,
		}
		return
	}

	vars["bid"] = botId.String

	if botType.String != "approved" {
		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   constants.NotApproved,
		}
		return
	}

	voteParsed, err := utils.GetVoteData(d.Context, vars["uid"], vars["bid"])

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
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

		d.Resp <- types.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   alreadyVotedMsg,
		}

		return
	}

	// Record new vote
	var itag pgtype.UUID
	err = state.Pool.QueryRow(d.Context, "INSERT INTO votes (user_id, bot_id) VALUES ($1, $2) RETURNING itag", vars["uid"], vars["bid"]).Scan(&itag)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	var oldVotes pgtype.Int4

	err = state.Pool.QueryRow(d.Context, "SELECT votes FROM bots WHERE bot_id = $1", vars["bid"]).Scan(&oldVotes)

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	var incr = 1
	var votes = oldVotes.Int32

	if utils.GetDoubleVote() {
		incr = 2
		votes += 2
	} else {
		votes++
	}

	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET votes = votes + $1 WHERE bot_id = $2", incr, vars["bid"])

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	userObj, err := utils.GetDiscordUser(vars["uid"])

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	botObj, err := utils.GetDiscordUser(vars["bid"])

	if err != nil {
		// Revert vote
		_, err := state.Pool.Exec(d.Context, "DELETE FROM votes WHERE itag = $1", itag)

		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	channel := os.Getenv("VOTE_LOGS_CHANNEL")

	state.Discord.ChannelMessageSendComplex(channel, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + vars["bid"],
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
						Value:  "[View " + botObj.Username + "](https://botlist.site/" + vars["bid"] + ")",
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[Vote for " + botObj.Username + "](https://botlist.site/" + vars["bid"] + "/vote)",
						Inline: true,
					},
				},
			},
		},
	})

	// Send webhook in a goroutine refunding the vote if it failed
	go func() {
		err = webhooks.Send(types.WebhookPost{
			BotID:  vars["bid"],
			UserID: vars["uid"],
			Votes:  int(votes),
		})

		var msg types.Message

		if err != nil {
			state.Pool.Exec(
				state.Context,
				"INSERT INTO alerts (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
				vars["uid"],
				"https://infinitybots.gg/bots/"+vars["bid"],
				"Whoa there! We've failed to notify this bot about this vote. The error was: "+err.Error()+".",
				"error",
			)
			msg = types.Message{
				Title:   "Whoa There!",
				Message: "Whoa there! We couldn't send " + botObj.Username + " this vote. The error was: " + err.Error() + ". Vote rewards may not work",
				Icon:    botObj.Avatar,
			}
		} else {
			state.Pool.Exec(
				state.Context,
				"INSERT INTO alerts (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
				vars["uid"],
				"https://infinitybots.gg/bots/"+vars["bid"],
				"state.Successfully alerted this bot to your vote with ID of "+vars["bid"]+"("+botObj.Username+")",
				"info",
			)
			msg = types.Message{
				Title:   "Vote Count Updated!",
				Message: "Successfully alerted " + botObj.Username + " to your vote with ID of " + vars["bid"] + ".",
			}
		}

		notifIds, err := state.Pool.Query(state.Context, "SELECT notif_id FROM poppypaw WHERE user_id = $1", vars["uid"])

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

			notifications.NotifChannel <- types.Notification{
				NotifID: notifId,
				Message: bytes,
			}
		}
	}()

	d.Resp <- types.HttpResponse{
		Status: http.StatusNoContent,
	}
}
