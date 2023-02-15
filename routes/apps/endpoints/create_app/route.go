package create_app

import (
	"net/http"
	"popplio/api"
	"popplio/apps"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/crypto"
)

type CreateApp struct {
	Position string            `json:"position" validate:"required"`
	Answers  map[string]string `json:"answers" validate:"required,dive,required"`
}

var compiledMessages = api.CompileValidationErrors(CreateApp{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create App For Position",
		Description: "Creates an application for a position. Returns a 204 on success.",
		Req:         CreateApp{},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to create the application for.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload CreateApp

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	position, ok := apps.Apps[payload.Position]

	if !ok {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "Invalid position",
			},
			Status: http.StatusBadRequest,
		}
	}

	if d.Auth.Banned && !position.AllowedForBanned {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "Banned users are not allowed to apply for this position",
			},
			Status: http.StatusBadRequest,
		}
	}

	if !d.Auth.Banned && position.BannedOnly {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are not banned? Why are you appealing?",
			},
			Status: http.StatusBadRequest,
		}
	}

	if position.Closed {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "This position is currently closed. Please check back later.",
			},
			Status: http.StatusBadRequest,
		}
	}

	var userApps int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(1) FROM apps WHERE user_id = $1 AND position = $2 AND state = 'pending'", d.Auth.ID, payload.Position).Scan(&userApps)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if userApps > 0 {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You already have a pending application for this position",
			},
			Status: http.StatusBadRequest,
		}
	}

	var answerMap = map[string]string{}

	for _, question := range position.Questions {
		ans, ok := payload.Answers[question.ID]

		if !ok {
			return api.HttpResponse{
				Json: types.ApiError{
					Error:   true,
					Message: "Missing answer for question " + question.ID,
				},
				Status: http.StatusBadRequest,
			}
		}

		if ans == "" {
			return api.HttpResponse{
				Json: types.ApiError{
					Error:   true,
					Message: "Answer for question " + question.ID + " cannot be empty",
				},
				Status: http.StatusBadRequest,
			}
		}

		if question.Short {
			if len(ans) > 4096 {
				return api.HttpResponse{
					Json: types.ApiError{
						Error:   true,
						Message: "Answer for question " + question.ID + " is too long",
					},
					Status: http.StatusBadRequest,
				}
			}
		} else {
			if len(ans) < 50 {
				return api.HttpResponse{
					Json: types.ApiError{
						Error:   true,
						Message: "Answer for question " + question.ID + " is too short",
					},
					Status: http.StatusBadRequest,
				}
			}

			if len(ans) > 10000 {
				return api.HttpResponse{
					Json: types.ApiError{
						Error:   true,
						Message: "Answer for question " + question.ID + " is too long",
					},
					Status: http.StatusBadRequest,
				}
			}
		}

		answerMap[question.ID] = ans
	}

	if position.ExtraLogic != nil {
		add, err := position.ExtraLogic(d, position, answerMap)

		if err != nil {
			state.Logger.Error(err)
			return api.HttpResponse{
				Json: types.ApiError{
					Error:   true,
					Message: "Error: " + err.Error(),
				},
				Status: http.StatusBadRequest,
			}
		}

		if !add {
			return api.DefaultResponse(http.StatusNoContent)
		}
	}

	var appId = crypto.RandString(64)

	_, err = state.Pool.Exec(
		d.Context,
		"INSERT INTO apps (app_id, user_id, position, questions, answers) VALUES ($1, $2, $3, $4, $5)",
		appId,
		d.Auth.ID,
		payload.Position,
		position.Questions,
		answerMap,
	)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Send a message to APPS channel
	var desc = "User <@" + d.Auth.ID + "> has applied for " + payload.Position + "."
	if position.PositionDescription != nil {
		desc = position.PositionDescription(d, position)
	}

	var channel = state.Config.Channels.Apps
	if position.Channel != nil {
		channel = position.Channel()
	}

	_, err = state.Discord.ChannelMessageSendComplex(channel, &discordgo.MessageSend{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "New " + position.Name + " Application!",
				URL:         state.Config.Sites.AppSite + "/panel/apps/" + appId,
				Description: desc,
				Color:       0x00ff00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "App ID",
						Value:  appId,
						Inline: true,
					},
					{
						Name:  "User ID",
						Value: d.Auth.ID,
					},
					{
						Name:   "Position",
						Value:  payload.Position,
						Inline: true,
					},
				},
			},
		},
	})

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
