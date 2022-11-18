package users

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"popplio/constants"
	"popplio/docs"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
	ua "github.com/mileusna/useragent"
	"go.uber.org/zap"
)

const tagName = "Users"

var (
	userColsArr = utils.GetCols(types.User{})
	userCols    = strings.Join(userColsArr, ",")

	silverpeltColsArr = utils.GetCols(types.Reminder{})
	silverpeltCols    = strings.Join(silverpeltColsArr, ",")

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to users on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/users", func(r chi.Router) {

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/users/{uid}/bots/{bid}/votes",
			OpId:        "get_user_votes",
			Summary:     "Get User Votes",
			Description: "Gets the users votes. **Requires authentication**",
			Tags:        []string{tagName},
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
			Resp: types.UserVote{
				Timestamps: []int64{},
				VoteTime:   12,
				HasVoted:   true,
			},
			AuthType: []string{"User", "Bot"},
		})
		r.Get("/{uid}/bots/{bid}/votes", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var vars = map[string]string{
					"uid": chi.URLParam(r, "uid"),
					"bid": chi.URLParam(r, "bid"),
				}

				userAuth := strings.HasPrefix(r.Header.Get("Authorization"), "User ")

				var botId pgtype.Text
				var botType pgtype.Text

				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				}

				var err error

				if userAuth {
					uid := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if uid == nil || *uid != vars["uid"] {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}

					err = state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId)

					if err != nil || !botId.Valid {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusNotFound)
						return
					}

					vars["bid"] = botId.String
				} else {
					err = state.Pool.QueryRow(ctx, "SELECT bot_id, type FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId, &botType)

					if err != nil || !botId.Valid || !botType.Valid {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusNotFound)
						return
					}

					id := utils.AuthCheck(r.Header.Get("Authorization"), true)

					if id == nil || *id != vars["bid"] {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}

					vars["bid"] = botId.String
				}

				voteParsed, err := utils.GetVoteData(ctx, vars["uid"], vars["bid"])

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				resp <- types.HttpResponse{
					Json: voteParsed,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "PUT",
			Path:        "/users/{uid}/bots/{bid}/votes",
			OpId:        "put_user_votes",
			Summary:     "Put User Votes",
			Description: "Posts a users votes. **For internal use only**",
			Tags:        []string{tagName},
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
			AuthType: []string{"User"},
		})
		r.Put("/{uid}/bots/{bid}/votes", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var vars = map[string]string{
					"uid": chi.URLParam(r, "uid"),
					"bid": chi.URLParam(r, "bid"),
				}

				if !strings.HasPrefix(r.Header.Get("Authorization"), "User ") {
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				var botId pgtype.Text
				var botType pgtype.Text

				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				}

				uid := utils.AuthCheck(r.Header.Get("Authorization"), false)

				if uid == nil || *uid != vars["uid"] {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				}

				var voteBannedState bool

				err := state.Pool.QueryRow(ctx, "SELECT vote_banned FROM users WHERE user_id = $1", uid).Scan(&voteBannedState)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if voteBannedState {
					resp <- types.HttpResponse{
						Status: http.StatusForbidden,
						Data:   constants.VoteBanned,
					}
					return
				}

				var voteBannedBotsState bool

				err = state.Pool.QueryRow(ctx, "SELECT bot_id, type, vote_banned FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId, &botType, &voteBannedBotsState)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if voteBannedBotsState {
					resp <- types.HttpResponse{
						Status: http.StatusForbidden,
						Data:   constants.VoteBanned,
					}
					return
				}

				vars["bid"] = botId.String

				if botType.String != "approved" {
					resp <- types.HttpResponse{
						Status: http.StatusBadRequest,
						Data:   constants.NotApproved,
					}
					return
				}

				voteParsed, err := utils.GetVoteData(ctx, vars["uid"], vars["bid"])

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
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

					resp <- types.HttpResponse{
						Status: http.StatusBadRequest,
						Json:   alreadyVotedMsg,
					}

					return
				}

				// Record new vote
				var itag pgtype.UUID
				err = state.Pool.QueryRow(ctx, "INSERT INTO votes (user_id, bot_id) VALUES ($1, $2) RETURNING itag", vars["uid"], vars["bid"]).Scan(&itag)

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				var oldVotes pgtype.Int4

				err = state.Pool.QueryRow(ctx, "SELECT votes FROM bots WHERE bot_id = $1", vars["bid"]).Scan(&oldVotes)

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
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

				_, err = state.Pool.Exec(ctx, "UPDATE bots SET votes = votes + $1 WHERE bot_id = $2", incr, vars["bid"])

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				userObj, err := utils.GetDiscordUser(vars["uid"])

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				botObj, err := utils.GetDiscordUser(vars["bid"])

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
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
							ctx,
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
							ctx,
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

				resp <- types.HttpResponse{
					Status: http.StatusNoContent,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/users/{id}/seo",
			OpId:        "get_user_seo",
			Summary:     "Get User SEO Info",
			Description: "Gets a users SEO data by id or username",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.SEO{},
			Tags: []string{tagName},
		})
		r.Get("/{id}/seo", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				name := chi.URLParam(r, "id")

				if name == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				cache := state.Redis.Get(ctx, "seou:"+name).Val()
				if cache != "" {
					resp <- types.HttpResponse{
						Data: cache,
						Headers: map[string]string{
							"X-Popplio-Cached": "true",
						},
					}
					return
				}

				var about string
				var userId string
				err := state.Pool.QueryRow(ctx, "SELECT about, user_id FROM users WHERE user_id = $1 OR username = $1", name).Scan(&about, &userId)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				user, err := utils.GetDiscordUser(userId)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				seo := types.SEO{
					ID:       user.ID,
					Username: user.Username,
					Avatar:   user.Avatar,
					Short:    about,
				}

				resp <- types.HttpResponse{
					Json:      seo,
					CacheKey:  "seou:" + name,
					CacheTime: 30 * time.Minute,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/users/notifications/info",
			OpId:        "get_user_notifications",
			Summary:     "Get User Notifications",
			Description: "Gets a users notifications",
			Resp:        types.NotificationInfo{},
			Tags:        []string{tagName},
		})
		r.Get("/notifications/info", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				data := types.NotificationInfo{
					PublicKey: os.Getenv("VAPID_PUBLIC_KEY"),
				}

				resp <- types.HttpResponse{
					Json: data,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/users/{id}/notifications",
			OpId:        "get_user_notifications",
			Summary:     "Get User Notifications",
			Description: "Gets a users notifications",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.NotifGetList{},
			Tags: []string{tagName},
		})
		r.Get("/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Fetch auth from postgresdb
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				var subscription []types.NotifGet

				var subscriptionDb []struct {
					Endpoint  string    `db:"endpoint"`
					NotifID   string    `db:"notif_id"`
					CreatedAt time.Time `db:"created_at"`
					UA        string    `db:"ua"`
				}

				rows, err := state.Pool.Query(ctx, "SELECT endpoint, notif_id, created_at, ua FROM poppypaw WHERE user_id = $1", id)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				err = pgxscan.ScanAll(&subscriptionDb, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if len(subscriptionDb) == 0 {
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				for _, sub := range subscriptionDb {
					uaD := ua.Parse(sub.UA)
					state.Logger.With(
						zap.String("endpoint", sub.Endpoint),
						zap.String("notif_id", sub.NotifID),
						zap.Time("created_at", sub.CreatedAt),
						zap.String("ua", sub.UA),
						zap.Any("browser", uaD),
					).Info("Parsed UA")

					binfo := types.NotifBrowserInfo{
						OS:         uaD.OS,
						Browser:    uaD.Name,
						BrowserVer: uaD.Version,
						Mobile:     uaD.Mobile,
					}

					subscription = append(subscription, types.NotifGet{
						Endpoint:    sub.Endpoint,
						NotifID:     sub.NotifID,
						CreatedAt:   sub.CreatedAt,
						BrowserInfo: binfo,
					})
				}

				sublist := types.NotifGetList{
					Notifications: subscription,
				}

				resp <- types.HttpResponse{
					Json: sublist,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "DELETE",
			Path:        "/users/{id}/notification",
			OpId:        "delete_user_notifications",
			Summary:     "Delete User Notifications",
			Description: "Deletes a users notification",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
				{
					Name:        "notif_id",
					Description: "Notification ID",
					Required:    true,
					In:          "query",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.ApiError{},
			Tags: []string{tagName},
		})
		r.Delete("/{id}/notification", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Check for notif_id
				if r.URL.Query().Get("notif_id") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Fetch auth from postgresdb
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				_, err := state.Pool.Exec(ctx, "DELETE FROM poppypaw WHERE user_id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id"))

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				resp <- types.HttpResponse{
					Status: http.StatusNoContent,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/users/{id}/reminders",
			OpId:        "get_user_reminders",
			Summary:     "Get User Reminders",
			Description: "Gets a users reminders",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.ReminderList{},
			Tags: []string{tagName},
		})
		r.Get("/{id}/reminders", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Fetch auth from postgresdb
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				// Fetch reminder from postgres
				rows, err := state.Pool.Query(ctx, "SELECT "+silverpeltCols+" FROM silverpelt WHERE user_id = $1", id)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				var reminders []types.Reminder

				pgxscan.ScanAll(&reminders, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if len(reminders) == 0 {
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				for i, reminder := range reminders {
					// Try resolving the bot from discord API
					var resolvedBot types.ResolvedReminderBot
					bot, err := utils.GetDiscordUser(reminder.BotID)

					if err != nil {
						resolvedBot = types.ResolvedReminderBot{
							Name:   "Unknown",
							Avatar: "https://cdn.discordapp.com/embed/avatars/0.png",
						}
					} else {
						resolvedBot = types.ResolvedReminderBot{
							Name:   bot.Username,
							Avatar: bot.Avatar,
						}
					}

					reminders[i].ResolvedBot = resolvedBot
				}

				reminderList := types.ReminderList{
					Reminders: reminders,
				}

				resp <- types.HttpResponse{
					Json: reminderList,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "DELETE",
			Path:        "/users/{id}/reminder",
			OpId:        "del_user_reminders",
			Summary:     "Delete User Reminders",
			Description: "Deletes a users reminders",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
				{
					Name:        "bot_id",
					Description: "Bot ID to delete a reminder of",
					Required:    true,
					In:          "query",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.ReminderList{},
			Tags: []string{tagName},
		})
		r.Delete("/{id}/reminder", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Fetch auth from postgres
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				var botId pgtype.Text

				err := state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", r.URL.Query().Get("bot_id")).Scan(&botId)

				if err != nil || !botId.Valid || botId.String == "" {
					state.Logger.Error("Error deleting reminder: ", err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				// Delete old
				state.Pool.Exec(ctx, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", id, botId.String)

				resp <- types.HttpResponse{
					Status: http.StatusNoContent,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "PUT",
			Path:        "/users/{id}/reminders",
			OpId:        "add_user_reminders",
			Summary:     "Add User Reminder",
			Description: "Deletes a users reminders",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
				{
					Name:        "bot_id",
					Description: "Bot ID to add a reminder of",
					Required:    true,
					In:          "query",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.ReminderList{},
			Tags: []string{tagName},
		})
		r.Put("/{id}/reminders", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Fetch auth from postgres
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				var botId pgtype.Text

				err := state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", r.URL.Query().Get("bot_id")).Scan(&botId)

				if err != nil || !botId.Valid || botId.String == "" {
					state.Logger.Error("Error adding reminder: ", err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				// Delete old
				state.Pool.Exec(ctx, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", id, botId.String)

				// Add new
				_, err = state.Pool.Exec(ctx, "INSERT INTO silverpelt (user_id, bot_id) VALUES ($1, $2)", id, botId.String)

				if err != nil {
					state.Logger.Error("Error adding reminder: ", err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				resp <- types.HttpResponse{
					Status: http.StatusNoContent,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "POST",
			Path:        "/users/{id}/sub",
			OpId:        "add_user_subscription",
			Summary:     "Add User Subscription",
			Description: "Adds a user subscription to a push notification",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Req:  types.UserSubscription{},
			Resp: types.ApiError{},
			Tags: []string{tagName},
		})
		r.Post("/{id}/sub", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var subscription types.UserSubscription

				var id = chi.URLParam(r, "id")

				if id == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				defer r.Body.Close()

				bodyBytes, err := io.ReadAll(r.Body)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				err = json.Unmarshal(bodyBytes, &subscription)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if subscription.Auth == "" || subscription.P256dh == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				// Fetch auth from postgres
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				// Store new subscription

				notifId := utils.RandString(512)

				ua := r.UserAgent()

				if ua == "" {
					ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
				}

				state.Pool.Exec(ctx, "DELETE FROM poppypaw WHERE user_id = $1 AND endpoint = $2", id, subscription.Endpoint)

				state.Pool.Exec(
					ctx,
					"INSERT INTO poppypaw (user_id, notif_id, auth, p256dh, endpoint, ua) VALUES ($1, $2, $3, $4, $5, $6)",
					id,
					notifId,
					subscription.Auth,
					subscription.P256dh,
					subscription.Endpoint,
					ua,
				)

				// Fan out test notification
				notifications.NotifChannel <- types.Notification{
					NotifID: notifId,
					Message: []byte(constants.TestNotif),
				}

				resp <- types.HttpResponse{
					Status: http.StatusNoContent,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "PATCH",
			Path:        "/users/{id}",
			OpId:        "update_user",
			Summary:     "Update User Profile",
			Description: "Updates a users profile",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Req:  types.ProfileUpdate{},
			Resp: types.ApiError{},
			Tags: []string{tagName},
		})
		r.Patch("/{id}", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				id := chi.URLParam(r, "id")

				// Fetch auth from postgresdb
				if r.Header.Get("Authorization") == "" {
					resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
					return
				} else {
					authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if authId == nil || *authId != id {
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}
				}

				// Fetch profile update from body
				var profile types.ProfileUpdate

				bodyBytes, err := io.ReadAll(r.Body)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				err = json.Unmarshal(bodyBytes, &profile)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				if profile.About != "" {
					if len(profile.About) > 1000 {
						resp <- types.HttpResponse{
							Status: http.StatusBadRequest,
							Data:   `{"error":true,"message": "About me is over 1000 characters!"}`,
						}
						return
					}

					// Update about
					_, err = state.Pool.Exec(ctx, "UPDATE users SET about = $1 WHERE user_id = $2", profile.About, id)

					if err != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
						return
					}
				}

				state.Redis.Del(ctx, "uc-"+id)

				resp <- types.HttpResponse{
					Status: http.StatusNoContent,
				}
			}()

			utils.Respond(ctx, w, resp)
		})

		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/users/{id}",
			OpId:        "get_user",
			Summary:     "Get User",
			Description: "Gets a user by id or username",
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "User ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.User{},
			Tags: []string{tagName},
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				name := chi.URLParam(r, "id")

				if name == "" {
					resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
					return
				}

				if name == "undefined" {
					resp <- types.HttpResponse{
						Status: http.StatusOK,
						Data:   `{"error":"false","message":"Handling known issue"}`,
					}
					return
				}

				// Check cache, this is how we can avoid hefty ratelimits
				cache := state.Redis.Get(ctx, "uc-"+name).Val()
				if cache != "" {
					resp <- types.HttpResponse{
						Data: cache,
						Headers: map[string]string{
							"X-Popplio-Cached": "true",
						},
					}
					return
				}

				var user types.User

				var err error

				row, err := state.Pool.Query(ctx, "SELECT "+userCols+" FROM users WHERE user_id = $1 OR username = $1", name)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				err = pgxscan.ScanOne(&user, row)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				user.ParseJSONB()

				err = utils.ParseUser(ctx, state.Pool, &user, state.Discord, state.Redis)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
					return
				}

				/* Removing or modifying fields directly in API is very dangerous as scrapers will
				 * just ignore owner checks anyways or cross-reference via another list. Also we
				 * want to respect the permissions of the owner if they're the one giving permission,
				 * blocking IPs is a better idea to this
				 */

				resp <- types.HttpResponse{
					Json:      user,
					CacheKey:  "uc-" + name,
					CacheTime: 3 * time.Minute,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}
