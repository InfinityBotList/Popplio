package edit_review

import (
	"net/http"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/events"
	"popplio/webhooks/serverhooks"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type EditReview struct {
	Content string `db:"content" json:"content" validate:"required,min=5,max=4000" msg:"Content must be between 5 and 4000 characters"`
	Stars   int32  `db:"stars" json:"stars" validate:"required,min=1,max=5" msg:"Stars must be between 1 and 5 stars"`
}

var compiledMessages = uapi.CompileValidationErrors(EditReview{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Review",
		Description: "Edits a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task and returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "rid",
				Description: "The review ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  EditReview{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload EditReview

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	rid := chi.URLParam(r, "rid")

	var author string
	var targetId string
	var targetType string
	var content string
	var stars int32

	err = state.Pool.QueryRow(d.Context, "SELECT author, target_id, target_type, content, stars FROM reviews WHERE id = $1", rid).Scan(&author, &targetId, &targetType, &content, &stars)

	if err != nil {
		state.Logger.Error("Failed to query review [db queryrow]", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if author != d.Auth.ID {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You are not the author of this review",
			},
		}
	}

	_, err = state.Pool.Exec(d.Context, "UPDATE reviews SET content = $1, stars = $2 WHERE id = $3", payload.Content, payload.Stars, rid)

	if err != nil {
		state.Logger.Error("Failed to update review", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	switch targetType {
	case "bot":
		err = bothooks.Send(bothooks.With{
			Data: events.WebhookBotEditReviewData{
				ReviewID: rid,
				Content: events.Changeset[string]{
					Old: content,
					New: payload.Content,
				},
				Stars: events.Changeset[int32]{
					Old: stars,
					New: payload.Stars,
				},
			},
			UserID: d.Auth.ID,
			BotID:  targetId,
		})

		if err != nil {
			state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("bot_id", targetId), zap.String("user_id", d.Auth.ID), zap.String("review_id", rid))
		}
	case "server":
		err = serverhooks.Send(serverhooks.With{
			Data: events.WebhookServerEditReviewData{
				ReviewID: rid,
				Content: events.Changeset[string]{
					Old: content,
					New: payload.Content,
				},
				Stars: events.Changeset[int32]{
					Old: stars,
					New: payload.Stars,
				},
			},
			UserID:   d.Auth.ID,
			ServerID: targetId,
		})

		if err != nil {
			state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("server_id", targetId), zap.String("user_id", d.Auth.ID), zap.String("review_id", rid))
		}
	default:
		state.Logger.Error("Unknown target type: " + targetType)
	}

	state.Redis.Del(d.Context, "rv-"+targetId+"-"+targetType)

	// Trigger a garbage collection step to remove any orphaned reviews
	go func() {
		err = assets.GCTrigger(targetId, targetType)

		if err != nil {
			state.Logger.Error("Failed to trigger GC: ", zap.Error(err))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
