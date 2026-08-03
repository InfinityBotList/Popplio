// Package add_review implements POST /{target_type}/{target_id}/reviews —
// "Create Review".
//
// Creates a new review for an entity. A user may have only one `root review`
// per entity. Triggers a garbage collection step to remove any orphaned
// reviews afterwards. Note that non-users can only create an 'owner review'.
// Returns 204 on success
package add_review

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
	"time"

	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.CreateReview{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Review",
		Description: "Creates a new review for an entity. A user may have only one `root review` per entity. Triggers a garbage collection step to remove any orphaned reviews afterwards. Note that non-users can only create an 'owner review'. Returns 204 on success",
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
		},
		Req:  types.CreateReview{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "review",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "review"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var payload types.CreateReview

	hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	switch targetType {
	case "bot":
		// Check if the bot exists
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", targetId).Scan(&count)

		if err != nil {
			return resp.Err("Failed to query bot count [db count]", err, zap.String("bot_id", targetId))
		}

		if count == 0 {
			return resp.BadRequest("Bot not found")
		}
	case "server":
		// Check if the server exists
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", targetId).Scan(&count)

		if err != nil {
			return resp.Err("Failed to query server count [db count]", err, zap.String("server_id", targetId))
		}

		if count == 0 {
			return resp.BadRequest("Server not found")
		}
	case "team":
		// Check if the team exists
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", targetId).Scan(&count)

		if err != nil {
			return resp.Err("Failed to query team count [db count]", err, zap.String("team_id", targetId))
		}

		if count == 0 {
			return resp.BadRequest("Team not found")
		}
	default:
		return resp.Status(http.StatusNotImplemented, "Support for this target type has not been implemented yet")
	}

	if d.Auth.TargetType != api.TargetTypeUser && !payload.OwnerReview {
		return resp.Forbidden("Only users may create non-owner reviews")
	}

	if payload.OwnerReview {
		// Perform entity specific checks
		err := api.AuthzEntityPermissionCheck(
			d.Context,
			d.Auth,
			targetType,
			targetId,
			perms.Permission{Namespace: targetType, Perm: teams.PermissionCreateOwnerReview},
		)

		if err != nil {
			return resp.Forbidden("Entity permission checks failed: " + err.Error())
		}
	}

	// Check if the user has already made a 'root' review for this entity
	if payload.ParentID == "" {
		var count int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reviews WHERE author = $1 AND target_id = $2 AND target_type = $3 AND parent_id IS NULL", d.Auth.ID, targetId, targetType).Scan(&count)

		if err != nil {
			return resp.Err("Failed to query root review count [db count]", err, zap.String("author", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
		}

		if count > 0 {
			return resp.Conflict("You have already made a root review for this " + targetType)
		}
	}

	// If parent_id is provided, check if it exists and check nesting
	if payload.ParentID != "" {
		var count int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reviews WHERE id = $1", payload.ParentID).Scan(&count)

		if err != nil {
			return resp.Err("Failed to query parent review count [db count]", err, zap.String("parent_id", payload.ParentID))
		}

		if count == 0 {
			return resp.BadRequest("Parent review not found")
		}

		nest, err := assets.Nest(d.Context, payload.ParentID)

		if err != nil {
			return resp.Err("Nesting engine failed unexpectedly", err, zap.String("parent_id", payload.ParentID))
		}

		if nest > 2 {
			return resp.BadRequest("Maximum nesting for reviews reached")
		}
	}

	// Create the review
	var parentId = pgtype.Text{
		Valid:  payload.ParentID != "",
		String: payload.ParentID,
	}

	var reviewId string
	err = state.Pool.QueryRow(d.Context, "INSERT INTO reviews (author, target_id, target_type, content, stars, parent_id, owner_review) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id", d.Auth.ID, targetId, targetType, payload.Content, payload.Stars, parentId, payload.OwnerReview).Scan(&reviewId)

	if err != nil {
		return resp.Err("Failed to insert review", err, zap.String("author", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
	}

	err = drivers.Send(drivers.With{
		Data: events.WebhookNewReviewData{
			ReviewID:    reviewId,
			Content:     payload.Content,
			Stars:       payload.Stars,
			OwnerReview: payload.OwnerReview,
		},
		UserID:     d.Auth.ID,
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType), zap.String("user_id", d.Auth.ID), zap.String("review_id", reviewId))
	}

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
