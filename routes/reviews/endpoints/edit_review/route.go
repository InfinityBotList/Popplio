package edit_review

import (
	"net/http"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/webhooks/core/drivers"
	cevents "popplio/webhooks/core/events"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.EditReview{})

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
		Req:  types.EditReview{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.EditReview

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
	var ownerReview bool

	err = state.Pool.QueryRow(d.Context, "SELECT author, target_id, target_type, content, stars, owner_review FROM reviews WHERE id = $1", rid).Scan(&author, &targetId, &targetType, &content, &stars, &ownerReview)

	if err != nil {
		state.Logger.Error("Failed to query review [db queryrow]", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if ownerReview {
		perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

		if err != nil {
			state.Logger.Error("Error getting entity perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
			}
		}

		if !perms.Has(targetType, teams.PermissionEditOwnerReview) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to edit an owner review for this " + targetType},
			}
		}
	} else {
		if author != d.Auth.ID {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json: types.ApiError{
					Message: "You are not the author of this review",
				},
			}
		}
	}

	_, err = state.Pool.Exec(d.Context, "UPDATE reviews SET content = $1, stars = $2 WHERE id = $3", payload.Content, payload.Stars, rid)

	if err != nil {
		state.Logger.Error("Failed to update review", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = drivers.Send(drivers.With{
		Data: events.WebhookEditReviewData{
			ReviewID:    rid,
			OwnerReview: ownerReview,
			Content: cevents.Changeset[string]{
				Old: content,
				New: payload.Content,
			},
			Stars: cevents.Changeset[int32]{
				Old: stars,
				New: payload.Stars,
			},
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
		err = assets.GCTrigger(targetId, targetType)

		if err != nil {
			state.Logger.Error("Failed to trigger GC: ", zap.Error(err))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
