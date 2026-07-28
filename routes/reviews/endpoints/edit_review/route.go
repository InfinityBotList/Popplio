package edit_review

import (
	"net/http"
	"popplio/api"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	cevents "popplio/webhooks/core/events"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.EditReview{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Review",
		Description: "Edits a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task. Note that non-users can only edit 'owner review's. Returns 204 on success",
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
		Req:  types.EditReview{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	rid := chi.URLParam(r, "review_id")

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

	var author string
	var content string
	var stars int32
	var ownerReview bool

	err = state.Pool.QueryRow(d.Context, "SELECT author, content, stars, owner_review FROM reviews WHERE id = $1 AND target_id = $2 AND target_type = $3", rid, targetId, targetType).Scan(&author, &content, &stars, &ownerReview)

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
			perms.Permission{Namespace: targetType, Perm: teams.PermissionEditOwnerReview},
		)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "Entity permission checks failed: " + err.Error()},
			}
		}
	} else {
		if d.Auth.TargetType != api.TargetTypeUser {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json: types.ApiError{
					Message: "Only users may edit non-owner reviews",
				},
			}
		} else if d.Auth.TargetType == api.TargetTypeUser {
			if author != d.Auth.ID {
				return uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json: types.ApiError{
						Message: "You are not the author of this review",
					},
				}
			}
		} else {
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json: types.ApiError{
					Message: "Unreachable condition reached!",
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
