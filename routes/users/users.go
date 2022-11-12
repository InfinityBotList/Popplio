package users

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgtype"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

const tagName = "Users"

var (
	userColsArr = utils.GetCols(types.User{})
	//
	userCols = strings.Join(userColsArr, ",")

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
		docs.Route(&docs.Doc{
			Method:      "POST",
			Path:        "/users/{uid}/bots/{bid}/votes",
			OpId:        "post_user_votes",
			Summary:     "Post User Votes",
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
		r.HandleFunc("/{uid}/bots/{bid}/votes", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" && r.Method != "PUT" {
				utils.ApiDefaultReturn(http.StatusMethodNotAllowed, w, r)
				return
			}

			var vars = map[string]string{
				"uid": chi.URLParam(r, "uid"),
				"bid": chi.URLParam(r, "bid"),
			}

			userAuth := strings.HasPrefix(r.Header.Get("Authorization"), "User ")

			var botId pgtype.Text
			var botType pgtype.Text

			if r.Header.Get("Authorization") == "" {
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			} else {
				var err error

				if userAuth {
					uid := utils.AuthCheck(r.Header.Get("Authorization"), false)

					if uid == nil || *uid != vars["uid"] {
						utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
						return
					}

					var voteBannedState bool

					err := state.Pool.QueryRow(state.Context, "SELECT vote_banned FROM users WHERE user_id = $1", uid).Scan(&voteBannedState)

					if err != nil {
						log.Error(err)
						utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
						return
					}

					if voteBannedState && r.Method == "PUT" {
						w.WriteHeader(http.StatusForbidden)
						w.Write([]byte(constants.VoteBanned))
						return
					}

					var voteBannedBotsState bool

					err = state.Pool.QueryRow(state.Context, "SELECT bot_id, type, vote_banned FROM bots WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", vars["bid"]).Scan(&botId, &botType, &voteBannedBotsState)

					if err != nil {
						log.Error(err)
						utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
						return
					}

					if voteBannedBotsState && r.Method == "PUT" {
						w.WriteHeader(http.StatusForbidden)
						w.Write([]byte(constants.VoteBanned))
						return
					}

					vars["bid"] = botId.String
				} else {
					err = state.Pool.QueryRow(state.Context, "SELECT bot_id, type FROM bots WHERE (vanity = $1 OR bot_id = $1 OR name = $1)", vars["bid"]).Scan(&botId, &botType)

					if err != nil || botId.Status != pgtype.Present || botType.Status != pgtype.Present {
						log.Error(err)
						utils.ApiDefaultReturn(http.StatusNotFound, w, r)
						return
					}

					vars["bid"] = botId.String

					id := utils.AuthCheck(r.Header.Get("Authorization"), true)

					if id == nil || *id != vars["bid"] {
						utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
						return
					}
				}
			}

			if botType.String != "approved" && r.Method == "PUT" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(constants.NotApproved))
				return
			}

			if !userAuth && r.Method == "PUT" {
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			voteParsed, err := utils.GetVoteData(state.Context, vars["uid"], vars["bid"])

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			if r.Method == "GET" {
				bytes, err := json.Marshal(voteParsed)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				w.Write(bytes)
			} else if r.Method == "PUT" {
				if voteParsed.HasVoted {
					timeElapsed := time.Now().UnixMilli() - voteParsed.LastVoteTime
					log.Info(timeElapsed)

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

					bytes, err := json.Marshal(alreadyVotedMsg)

					if err != nil {
						log.Error(err)
						utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
						return
					}

					w.WriteHeader(http.StatusBadRequest)
					w.Write(bytes)
					return
				}

				// Record new vote
				var itag pgtype.UUID
				err := state.Pool.QueryRow(state.Context, "INSERT INTO votes (user_id, bot_id) VALUES ($1, $2) RETURNING itag", vars["uid"], vars["bid"]).Scan(&itag)

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(state.Context, "DELETE FROM votes WHERE itag = $1", itag)
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				var oldVotes pgtype.Int4

				err = state.Pool.QueryRow(state.Context, "SELECT votes FROM bots WHERE bot_id = $1", vars["bid"]).Scan(&oldVotes)

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(state.Context, "DELETE FROM votes WHERE itag = $1", itag)

					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				var incr = 1
				var votes = oldVotes.Int

				if utils.GetDoubleVote() {
					incr = 2
					votes += 2
				} else {
					votes++
				}

				_, err = state.Pool.Exec(state.Context, "UPDATE bots SET votes = votes + $1 WHERE bot_id = $2", incr, vars["bid"])

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(state.Context, "DELETE FROM votes WHERE itag = $1", itag)

					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				userObj, err := utils.GetDiscordUser(vars["uid"])

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(state.Context, "DELETE FROM votes WHERE itag = $1", itag)

					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				botObj, err := utils.GetDiscordUser(vars["bid"])

				if err != nil {
					// Revert vote
					_, err := state.Pool.Exec(state.Context, "DELETE FROM votes WHERE itag = $1", itag)

					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
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

					if err != nil {
						state.Pool.Exec(
							state.Context,
							"INSERT INTO notifications (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
							vars["uid"],
							"https://infinitybots.gg/bots/"+vars["bid"],
							"Whoa there! We've failed to notify this bot about this vote. The error was: "+err.Error()+".",
							"error")
					} else {
						state.Pool.Exec(
							state.Context,
							"INSERT INTO notifications (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
							vars["uid"],
							"https://infinitybots.gg/bots/"+vars["bid"],
							"state.Successfully voted for bot with ID of "+vars["bid"],
							"info",
						)
					}
				}()

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(constants.Success))
			}
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
			name := chi.URLParam(r, "id")

			if name == "" {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			cache := state.Redis.Get(state.Context, "seou:"+name).Val()
			if cache != "" {
				w.Header().Add("X-Popplio-Cached", "true")
				w.Write([]byte(cache))
				return
			}

			var about string
			var userId string
			err := state.Pool.QueryRow(state.Context, "SELECT about, user_id FROM users WHERE user_id = $1 OR username = $1", name).Scan(&about, &userId)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			user, err := utils.GetDiscordUser(userId)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			bytes, err := json.Marshal(types.SEO{
				ID:       user.ID,
				Username: user.Username,
				Avatar:   user.Avatar,
				Short:    about,
			})

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			state.Redis.Set(state.Context, "seou:"+name, string(bytes), time.Minute*30)

			w.Write(bytes)
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
			Resp: types.User{},
			Tags: []string{tagName},
		})
		r.Patch("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")

			// Fetch auth from postgresdb
			if r.Header.Get("Authorization") == "" {
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			} else {
				authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

				if authId == nil || *authId != id {
					utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
					return
				}
			}

			// Fetch profile update from body
			var profile types.ProfileUpdate

			bodyBytes, err := io.ReadAll(r.Body)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			err = json.Unmarshal(bodyBytes, &profile)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			if profile.About != "" {
				if len(profile.About) > 1000 {
					w.Write([]byte(`{"error":true,"message": "About me is over 1000 characters!"}`))
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Update about
				_, err = state.Pool.Exec(state.Context, "UPDATE users SET about = $1 WHERE user_id = $2", profile.About, id)

				if err != nil {
					log.Error(err)
					utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}
			}

			state.Redis.Del(state.Context, "uc-"+id)
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
			name := chi.URLParam(r, "id")

			if name == "" {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			if name == "undefined" {
				w.Write([]byte(`{"error":"false","message":"Handling known issue"}`))
				return
			}

			// Check cache, this is how we can avoid hefty ratelimits
			cache := state.Redis.Get(state.Context, "uc-"+name).Val()
			if cache != "" {
				w.Header().Add("X-Popplio-Cached", "true")
				w.Write([]byte(cache))
				return
			}

			var user types.User

			var err error

			row, err := state.Pool.Query(state.Context, "SELECT "+userCols+" FROM users WHERE user_id = $1 OR username = $1", name)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			err = pgxscan.ScanOne(&user, row)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			err = utils.ParseUser(state.Context, state.Pool, &user, state.Discord, state.Redis)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			/* Removing or modifying fields directly in API is very dangerous as scrapers will
			 * just ignore owner checks anyways or cross-reference via another list. Also we
			 * want to respect the permissions of the owner if they're the one giving permission,
			 * blocking IPs is a better idea to this
			 */

			bytes, err := json.Marshal(user)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			state.Redis.Set(state.Context, "uc-"+name, string(bytes), time.Minute*3)

			w.Write(bytes)
		})
	})
}
