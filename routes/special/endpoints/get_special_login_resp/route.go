package get_special_login_resp

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"popplio/api"
	"popplio/routes/special/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"

	docs "github.com/infinitybotlist/doclib"

	_ "embed"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/crypto"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//go:embed templates/confirm.html
var confirmTemplate string

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Special Login Handler",
		Description: "This endpoint is used to respond to a special login. It then spawns the task such as data requests etc.",
		Resp:        "[Redirect+Task Creation]",
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	stateQuery := r.URL.Query().Get("state")

	// Get act from redis
	act, err := state.Redis.Get(d.Context, "spec:"+stateQuery).Result()

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
	}

	// Decode act using json
	var action assets.Action

	err = json.Unmarshal([]byte(act), &action)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state",
		}
	}

	// Check time
	if time.Since(action.Time) > 3*time.Minute {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid state (too old). Please retry what you were doing!",
		}
	}

	// Ask for confirmation from the user
	if action.Nonce == "" || r.URL.Query().Get("n") == "" || r.Header.Get("X-High-Security-Mode") != "true" {
		confirmNonce := crypto.RandString(96)

		// Set confirm nonce in redis
		action.Nonce = confirmNonce

		// Set action in redis
		rBytes, err := json.Marshal(action)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}
		state.Redis.Set(d.Context, "spec:"+stateQuery, rBytes, 3*time.Minute)

		// Send confirmation page
		templateResp := bytes.Buffer{}

		var prettyName = "Unknown"

		switch action.Action {
		case "dr":
			prettyName = "Data Request"
		case "ddr":
			prettyName = "Data Deletion Request"
		case "rtu":
			prettyName = "Reset User Token"
		case "rtb":
			prettyName = "Reset Bot Token"
		case "bhmac":
			prettyName = "Bot HMAC Update"
		case "bweburl":
			prettyName = "Bot Webhook URL Update"
		case "bwebsec":
			prettyName = "Bot Webhook Secret Update"
		case "db":
			prettyName = "Delete Bot"
		case "tb":
			prettyName = "Transfer Bot Ownership"
		}

		err = template.Must(template.New("confirm").Parse(confirmTemplate)).Execute(&templateResp, assets.ConfirmTemplate{
			Action:       action,
			PrettyAction: prettyName,
			RandPhrase:   crypto.RandString(8),
		})

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Status: http.StatusOK,
			Bytes:  templateResp.Bytes(),
			Headers: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
		}
	}

	state.Redis.Del(d.Context, "spec:"+stateQuery)

	if r.URL.Query().Get("n") != action.Nonce {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid nonce",
		}
	}

	// Check code with discords api
	data := url.Values{}

	data.Set("client_id", state.Config.HighSecurityCtx.ClientID)
	data.Set("client_secret", state.Config.HighSecurityCtx.ClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("redirect_uri", state.Config.HighSecurityCtx.RedirectURL)

	response, err := http.PostForm("https://discord.com/api/oauth2/token", data)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	var token struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	err = json.Unmarshal(body, &token)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	state.Logger.Info(token)

	if !strings.Contains(token.Scope, "identify") {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid scope: scope contain identify, is currently " + token.Scope,
		}
	}

	// Get user info
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: time.Second * 10}

	response, err = client.Do(req)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	var user types.OauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	var perms *teams.PermissionManager

	if action.TID != "" {
		// Validate that they actually own this bot
		perms, err = utils.GetUserBotPerms(d.Context, user.ID, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   "Bot check failed:" + err.Error(),
			}
		}

		if !perms.HasSomePerms() {
			return api.HttpResponse{
				Status: http.StatusUnauthorized,
				Data:   "You cannot access high security actions for this bot",
			}
		}
	}

	switch action.Action {
	// Data request
	case "dr":
		taskId := crypto.RandString(196)

		err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go assets.DataTask(taskId, user.ID, remoteIp[0], false)

		return api.HttpResponse{
			Headers: map[string]string{
				"X-Defer": taskId,
			},
		}
	// Data deletion request
	case "ddr":
		taskId := crypto.RandString(196)

		err = state.Redis.Set(d.Context, taskId, "WAITING", time.Hour*8).Err()

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go assets.DataTask(taskId, user.ID, remoteIp[0], true)
		return api.HttpResponse{
			Headers: map[string]string{
				"X-Defer": taskId,
			},
		}
	// Reset token for users
	case "rtu":
		var token string
		token = crypto.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Data: "Your new API token is: " + token + "\n\nThank you and have a nice day ;)",
		}
	// Reset token for bots
	case "rtb":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if !perms.Has(teams.TeamPermissionResetBotTokens) {
			return api.HttpResponse{
				Status: http.StatusUnauthorized,
				Data:   "You do not have permission to reset this bot's token",
			}
		}

		token := crypto.RandString(128)

		_, err := state.Pool.Exec(d.Context, "UPDATE bots SET api_token = $1 WHERE bot_id = $2", token, action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		return api.HttpResponse{
			Data: "Your new API token is: " + token + "\n\nThank you and have a nice day ;)",
		}
	// Set HMAC secret for bots
	case "bhmac":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if !perms.Has(teams.TeamPermissionEditBotWebhooks) {
			return api.HttpResponse{
				Status: http.StatusUnauthorized,
				Data:   "You do not have permission to edit this bot's hmac secret",
			}
		}

		if action.Ctx != "true" && action.Ctx != "false" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "Invalid value for hmac",
			}
		}

		utils.ClearBotCache(d.Context, action.TID)

		if action.Ctx == "true" {
			// We want to unset hmac
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET hmac = true WHERE bot_id = $1", action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully set hmac",
			}
		} else {
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET hmac = false WHERE bot_id = $1", action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully unset hmac",
			}
		}
	// Bot webhook url update
	case "bweburl":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if !perms.Has(teams.TeamPermissionEditBotWebhooks) {
			return api.HttpResponse{
				Status: http.StatusUnauthorized,
				Data:   "You do not have permission to edit this bot's webhook url",
			}
		}

		if action.Ctx == "" {
			// We want to unset webhook secret
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully unset webhook url",
			}
		} else {
			// Ensure webhook is a URL
			if !strings.HasPrefix(action.Ctx, "https://") && action.Ctx != "httpUser" {
				return api.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid webhook url",
				}
			}

			if webhooks.IsDiscordURL(action.Ctx) {
				return api.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Discord webhooks are not supported at this time due to connection issues and abuse. See https://github.com/infinitybotlist/iblcli for the alternative solution.",
				}
			}

			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET webhook = $1 WHERE bot_id = $2", action.Ctx, action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully set webhook url",
			}
		}
	// Bot webhook secret update
	case "bwebsec":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if !perms.Has(teams.TeamPermissionEditBotWebhooks) {
			return api.HttpResponse{
				Status: http.StatusUnauthorized,
				Data:   "You do not have permission to edit this bot's webhook secret",
			}
		}

		if action.Ctx == "" {
			// We want to unset webhook secret
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET web_auth = NULL WHERE bot_id = $1", action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully unset webhook secret",
			}
		} else {
			_, err := state.Pool.Exec(d.Context, "UPDATE bots SET web_auth = $1 WHERE bot_id = $2", action.Ctx, action.TID)

			if err != nil {
				return api.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
			}

			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully set webhook secret",
			}
		}
	// Delete the bot
	case "db":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		if !perms.Has(teams.TeamPermissionDeleteBots) {
			return api.HttpResponse{
				Status: http.StatusUnauthorized,
				Data:   "You do not have permission to delete bots",
			}
		}

		// Clear cache
		utils.ClearBotCache(d.Context, action.TID)

		// Delete bot
		_, err = state.Pool.Exec(d.Context, "DELETE FROM bots WHERE bot_id = $1", action.TID)

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Data:   err.Error(),
			}
		}

		// Send embed to bot log channel
		_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
			Content: "",
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:   state.Config.Sites.Frontend + "/bots/" + action.TID,
					Title: "Bot Deleted",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Bot ID",
							Value: action.TID,
						},
						{
							Name:  "Deleter",
							Value: fmt.Sprintf("<@%s>", user.ID),
						},
					},
				},
			},
		})

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusOK,
				Data:   "Successfully deleted bot [ :) ] but we couldn't send a log message [ :( ]",
			}
		}

		return api.HttpResponse{
			Status: http.StatusOK,
			Data:   "Successfully deleted bot :)",
		}
	// Transfer bot ownership
	case "tb":
		if action.TID == "" {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Data:   "No target id set",
			}
		}

		return api.HttpResponse{
			Status: http.StatusOK,
			Data:   "This is currently disabled while we transfer to teams! Please contact us if you need to transfer a bot at this time",
		}

	default:
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   "Invalid action",
		}
	}
}
