package add_review

import (
	"net/http"
	"popplio/api"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/events"
	"time"

	"popplio/api/authz"

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
		state.Logger.Error("Error while ratelimiting", zap.Error(err), zap.String("bucket", "review"))
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
			state.Logger.Error("Failed to query bot count [db count]", zap.Error(err), zap.String("bot_id", targetId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Bot not found"},
			}
		}
	case "server":
		// Check if the server exists
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error("Failed to query server count [db count]", zap.Error(err), zap.String("server_id", targetId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Server not found"},
			}
		}
	case "team":
		// Check if the team exists
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error("Failed to query team count [db count]", zap.Error(err), zap.String("team_id", targetId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Team not found"},
			}
		}
	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "Support for this target type has not been implemented yet"},
		}
	}

	if d.Auth.TargetType != api.TargetTypeUser && !payload.OwnerReview {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "Only users may create non-owner reviews"},
		}
	}

	if payload.OwnerReview {
		// Perform entity specific checks
		err := authz.EntityPermissionCheck(
			d.Context,
			d.Auth,
			targetType,
			targetId,
			perms.Permission{Namespace: targetType, Perm: teams.PermissionCreateOwnerReview},
		)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "Entity permission checks failed: " + err.Error()},
			}
		}
	}

	// Check if the user has already made a 'root' review for this entity
	if payload.ParentID == "" {
		var count int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reviews WHERE author = $1 AND target_id = $2 AND target_type = $3 AND parent_id IS NULL", d.Auth.ID, targetId, targetType).Scan(&count)

		if err != nil {
			state.Logger.Error("Failed to query root review count [db count]", zap.Error(err), zap.String("author", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count > 0 {
			return uapi.HttpResponse{
				Status: http.StatusConflict,
				Json:   types.ApiError{Message: "You have already made a root review for this " + targetType},
			}
		}
	}

	// If parent_id is provided, check if it exists and check nesting
	if payload.ParentID != "" {
		var count int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reviews WHERE id = $1", payload.ParentID).Scan(&count)

		if err != nil {
			state.Logger.Error("Failed to query parent review count [db count]", zap.Error(err), zap.String("parent_id", payload.ParentID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Parent review not found"},
			}
		}

		nest, err := assets.Nest(d.Context, payload.ParentID)

		if err != nil {
			state.Logger.Error("Nesting engine failed unexpectedly", zap.Error(err), zap.String("parent_id", payload.ParentID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if nest > 2 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Maximum nesting for reviews reached"},
			}
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
		state.Logger.Error("Failed to insert review", zap.Error(err), zap.String("author", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
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
