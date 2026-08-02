// Package remove_review implements DELETE
// /{target_type}/{target_id}/reviews/{review_id} — "Delete Review".
//
// Deletes a review by review ID. The user must be the author of this review.
// This will automatically trigger a garbage collection task and returns 204
// on success
package remove_review

import (
	"net/http"
	"popplio/api"
	"popplio/api/resp"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Review",
		Description: "Deletes a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task and returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "review_id",
				Description: "The review ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	rid := chi.URLParam(r, "review_id")

	var author string
	var content string
	var stars int32
	var ownerReview bool

	err := state.Pool.QueryRow(d.Context, "SELECT author, content, stars, owner_review FROM reviews WHERE id = $1 AND target_id = $2 AND target_type = $3", rid, targetId, targetType).Scan(&author, &content, &stars, &ownerReview)

	if err != nil {
		state.Logger.Error("Failed to query review [db queryrow]", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if ownerReview {
		// Perform entity specific checks
		err := api.AuthzEntityPermissionCheck(
			d.Context,
			d.Auth,
			targetType,
			targetId,
			perms.Permission{Namespace: targetType, Perm: teams.PermissionDeleteOwnerReview},
		)

		if err != nil {
			return resp.Forbidden("Entity permission checks failed: " + err.Error())
		}
	} else {
		if d.Auth.TargetType != api.TargetTypeUser {
			return resp.Forbidden("Only users may delete non-owner reviews")
		} else if d.Auth.TargetType == api.TargetTypeUser {
			if author != d.Auth.ID {
				return resp.Forbidden("You are not the author of this review")
			}
		} else {
			return resp.ErrBody("Unreachable condition reached!", "Unreachable condition reached!", nil)
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM reviews WHERE id = $1", rid)

	if err != nil {
		return resp.Err("Failed to delete review [db exec]", err, zap.String("rid", rid))
	}

	err = drivers.Send(drivers.With{
		Data: events.WebhookDeleteReviewData{
			ReviewID:    rid,
			Content:     content,
			Stars:       stars,
			OwnerReview: ownerReview,
		},
		UserID:     d.Auth.ID,
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType), zap.String("user_id", d.Auth.ID), zap.String("review_id", rid))
	}

	state.Redis.Del(d.Context, "rv-"+targetId+"-"+targetType)

	// Trigger a garbage collection step to remove any orphaned reviews
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while triggering review GC", zap.Any("panic", rec), zap.String("target_id", targetId), zap.String("target_type", targetType))
			}
		}()

		if err := assets.GCTrigger(targetId, targetType); err != nil {
			state.Logger.Error("Failed to trigger GC: ", zap.Error(err))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
