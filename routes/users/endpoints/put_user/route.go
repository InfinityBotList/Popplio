package put_user

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/ratelimit"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var (
	json             = jsoniter.ConfigCompatibleWithStandardLibrary
	compiledMessages = uapi.CompileValidationErrors(types.AuthorizeRequest{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Login User",
		Description: "Takes in a ``code`` query parameter and returns a user ``token``. **Cannot be used outside of the site for security reasons but documented in case we wish to allow its use in the future.**",
		Req:         types.AuthorizeRequest{},
		Resp:        types.UserLogin{},
	}
}

// OauthInfo struct for oauth2 info
type oauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
}

func sendAuthLog(user oauthUser, req types.AuthorizeRequest, new bool) {
	var banned bool
	var voteBanned bool

	if !new {
		err := state.Pool.QueryRow(state.Context, "SELECT banned, vote_banned FROM users WHERE user_id = $1", user.ID).Scan(&banned, &voteBanned)

		if err != nil {
			state.Logger.Error("sendAuthLog: Failed to get user details from database", zap.Error(err), zap.String("user_id", user.ID))
			return
		}
	}

	state.Logger.With(
		zap.String("user_id", user.ID),
		zap.String("channel_id", state.Config.Channels.AuthLogs),
		zap.String("bot_info", state.Discord.State.User.String()),
	).Debug("sendAuthLog: Channel Info")

	_, err := state.Discord.ChannelMessageSendComplex(state.Config.Channels.AuthLogs, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title: "User Login Attempt",
				Color: 0xff0000,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User ID",
						Value:  user.ID,
						Inline: true,
					},
					{
						Name:   "Username",
						Value:  user.Username + "#" + user.Disc,
						Inline: true,
					},
					{
						Name:   "Scope",
						Value:  req.Scope,
						Inline: true,
					},
					{
						Name: "Banned",
						Value: func() string {
							if banned {
								return "Yes"
							}

							return "No"
						}(),
						Inline: true,
					},
					{
						Name: "Vote Banned",
						Value: func() string {
							if voteBanned {
								return "Yes"
							}

							return "No"
						}(),
						Inline: true,
					},
					{
						Name: "New User",
						Value: func() string {
							if new {
								return "Yes"
							}

							return "No"
						}(),
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "© Copyright 2023 - Infinity Bot List",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	})

	if err != nil {
		state.Logger.Error("sendAuthLog: Failed to send message to Discord", zap.Error(err), zap.String("user_id", user.ID))
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "login",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error("Error while ratelimiting", zap.Error(err), zap.String("bucket", "login"))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var req types.AuthorizeRequest

	hresp, ok := uapi.MarshalReqWithHeaders(r, &req, limit.Headers())

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(req)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if req.Protocol != "persepolis" {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Your client is outdated and is not supported. Please contact the developers of this client.",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if !slices.Contains(state.Config.DiscordAuth.AllowedRedirects, req.RedirectURI) {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Malformed redirect_uri",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if req.ClientID != state.Config.DiscordAuth.ClientID {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Misconfigured client! Client id is incorrect",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if state.Redis.Exists(d.Context, "codecache:"+req.Code).Val() == 1 {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Code has been clearly used before and is as such invalid",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	state.Redis.Set(d.Context, "codecache:"+req.Code, "0", 5*time.Minute)

	httpResp, err := http.PostForm("https://discord.com/api/v10/oauth2/token", url.Values{
		"client_id":     {state.Config.DiscordAuth.ClientID},
		"client_secret": {state.Config.DiscordAuth.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {req.Code},
		"redirect_uri":  {req.RedirectURI},
		"scope":         {"identify"},
	})

	if err != nil {
		state.Logger.Error("Failed to send oauth2 token request to discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to send token request to Discord",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)

	if err != nil {
		state.Logger.Error("Failed to read oauth2 token response from discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to read token response from Discord",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}

	err = json.Unmarshal(body, &token)

	if err != nil {
		state.Logger.Error("Failed to parse oauth2 token response from discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to parse token response from Discord",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	if token.AccessToken == "" {
		state.Logger.Error("No access token provided by discord")
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "No access token provided by Discord",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	cli := &http.Client{}

	var httpReq *http.Request
	httpReq, err = http.NewRequestWithContext(d.Context, "GET", "https://discord.com/api/v10/users/@me", nil)

	if err != nil {
		state.Logger.Error("Failed to create oauth2 request to discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to create request to Discord to fetch user info",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	httpResp, err = cli.Do(httpReq)

	if err != nil {
		state.Logger.Error("Failed to send oauth2 request to discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to send oauth2 request to Discord",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	defer httpResp.Body.Close()

	body, err = io.ReadAll(httpResp.Body)

	if err != nil {
		state.Logger.Error("Failed to read oauth2 response from discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to read oauth2 response from Discord",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	var user oauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		state.Logger.Error("Failed to parse oauth2 response from discord", zap.Error(err))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to parse oauth2 response from Discord",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	if user.ID == "" {
		state.Logger.Error("No user ID provided by discord. Invalid code/access token?")
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "No user ID provided by Discord. Invalid code/access token?",
			},
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
		}
	}

	// Check if user exists on database
	var exists bool

	err = state.Pool.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", user.ID).Scan(&exists)

	if err != nil {
		state.Logger.Error("Failed to check if user exists on database", zap.Error(err), zap.String("userID", user.ID))
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "Failed to check if user exists on database",
			},
			Status:  http.StatusInternalServerError,
			Headers: limit.Headers(),
		}
	}

	var apiToken string

	if !exists {
		// Create user
		apiToken = crypto.RandString(128)
		_, err = state.Pool.Exec(
			d.Context,
			"INSERT INTO users (user_id, api_token, extra_links, developer, certified) VALUES ($1, $2, $3, false, false)",
			user.ID,
			apiToken,
			[]types.Link{},
		)

		if err != nil {
			state.Logger.Error("Failed to create user on database", zap.Error(err), zap.String("userID", user.ID))
			return uapi.HttpResponse{
				Json: types.ApiError{
					Message: "Failed to create user on database",
				},
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
			}
		}

		go sendAuthLog(user, req, true)
	} else {
		// Get API token and ban state
		var banned bool
		var tokenStr pgtype.Text

		err = state.Pool.QueryRow(d.Context, "SELECT banned, api_token FROM users WHERE user_id = $1", user.ID).Scan(&banned, &tokenStr)

		if err != nil {
			state.Logger.Error("Failed to get API token from database", zap.Error(err), zap.String("userID", user.ID))
			return uapi.HttpResponse{
				Json: types.ApiError{
					Message: "Failed to get API token from database",
				},
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
			}
		}

		go sendAuthLog(user, req, false)

		if banned && req.Scope != "ban_exempt" {
			return uapi.HttpResponse{
				Json: types.ApiError{
					Message: "You are banned from the list. If you think this is a mistake, please contact support.",
				},
				Status:  http.StatusForbidden,
				Headers: limit.Headers(),
			}
		}

		if !banned && req.Scope == "ban_exempt" {
			return uapi.HttpResponse{
				Json: types.ApiError{
					Message: "The selected scope is not allowed for unbanned users [ban_exempt].",
				},
				Status:  http.StatusForbidden,
				Headers: limit.Headers(),
			}
		}

		apiToken = tokenStr.String
	}

	// Create authUser and send
	var authUser = types.UserLogin{
		UserID: user.ID,
		Token:  apiToken,
	}

	return uapi.HttpResponse{
		Json:    authUser,
		Headers: limit.Headers(),
	}
}
