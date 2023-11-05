package remove_review

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
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Review",
		Description: "Deletes a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task and returns 204 on success",
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
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rid := chi.URLParam(r, "rid")

	var author string
	var targetId string
	var targetType string
	var content string
	var stars int32

	err := state.Pool.QueryRow(d.Context, "SELECT author, target_id, target_type, content, stars FROM reviews WHERE id = $1", rid).Scan(&author, &targetId, &targetType, &content, &stars)

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

	_, err = state.Pool.Exec(d.Context, "DELETE FROM reviews WHERE id = $1", rid)

	if err != nil {
		state.Logger.Error("Failed to delete review [db exec]", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	switch targetType {
	case "bot":
		err = bothooks.Send(bothooks.With{
			Data: events.WebhookBotDeleteReviewData{
				ReviewID: rid,
				Content:  content,
				Stars:    stars,
			},
			UserID: d.Auth.ID,
			BotID:  targetId,
		})

		if err != nil {
			state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("bot_id", targetId), zap.String("user_id", d.Auth.ID), zap.String("review_id", rid))
		}
	case "server":
		err = serverhooks.Send(serverhooks.With{
			Data: events.WebhookServerDeleteReviewData{
				ReviewID: rid,
				Content:  content,
				Stars:    stars,
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
