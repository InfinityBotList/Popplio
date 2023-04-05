package add_bot_review

import (
	"net/http"
	"popplio/api"
	"popplio/ratelimit"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/events"
	"strings"
	"time"

	docs "github.com/infinitybotlist/doclib"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type CreateReview struct {
	Content  string `db:"content" json:"content" validate:"required,min=5,max=4000" msg:"Content must be between 5 and 4000 characters"`
	Stars    int32  `db:"stars" json:"stars" validate:"required,min=1,max=5" msg:"Stars must be between 1 and 5 stars"`
	ParentID string `db:"parent_id" json:"parent_id" validate:"omitempty,uuid" msg:"Parent ID must be a valid UUID if provided"`
}

var (
	compiledMessages = api.CompileValidationErrors(CreateReview{})

	reviewColsArr = utils.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Bot Review",
		Description: "Creates a new users review of a bot. A user may have only one `root review` per bot. Triggers a garbage collection step to remove any orphaned reviews afterwards. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "The bots ID or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  CreateReview{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "review",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var payload CreateReview

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	botId := chi.URLParam(r, "bid")

	bot, err := utils.ResolveBot(d.Context, botId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if bot == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Check if the user has already made a 'root' review for this bot
	if payload.ParentID == "" {
		var count int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reviews WHERE author = $1 AND bot_id = $2 AND parent_id IS NULL", d.Auth.ID, bot).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		if count > 0 {
			return api.HttpResponse{
				Status: http.StatusConflict,
				Json: types.ApiError{
					Message: "You have already made a root review for this bot",
					Error:   true,
				},
			}
		}
	}

	// If parent_id is provided, check if it exists and check nesting
	if payload.ParentID != "" {
		var count int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reviews WHERE id = $1", payload.ParentID).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "Parent review not found",
					Error:   true,
				},
			}
		}

		if assets.Nest(d.Context, payload.ParentID) > 2 {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "Maximum nesting for reviews reached",
					Error:   true,
				},
			}
		}
	}

	// Create the review
	var reviewId string
	if payload.ParentID == "" {
		err = state.Pool.QueryRow(d.Context, "INSERT INTO reviews (author, bot_id, content, stars) VALUES ($1, $2, $3, $4) RETURNING id", d.Auth.ID, bot, payload.Content, payload.Stars).Scan(&reviewId)
	} else {
		err = state.Pool.QueryRow(d.Context, "INSERT INTO reviews (author, bot_id, content, stars, parent_id) VALUES ($1, $2, $3, $4, $5) RETURNING id", d.Auth.ID, bot, payload.Content, payload.Stars, payload.ParentID).Scan(&reviewId)
	}

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = bothooks.Send(bothooks.With[events.WebhookBotNewReviewData]{
		Data: events.WebhookBotNewReviewData{
			ReviewID: reviewId,
			Content:  payload.Content,
		},
		UserID: d.Auth.ID,
		BotID:  bot,
	})

	if err != nil {
		state.Logger.Error(err)
	}

	state.Redis.Del(d.Context, "rv-"+bot)

	// Trigger a garbage collection step to remove any orphaned reviews
	go func() {
		rows, err := state.Pool.Query(state.Context, "SELECT "+reviewCols+" FROM reviews WHERE bot_id = $1 ORDER BY created_at ASC", bot)

		if err != nil {
			state.Logger.Error(err)
		}

		var reviews []types.Review = []types.Review{}

		err = pgxscan.ScanAll(&reviews, rows)

		if err != nil {
			state.Logger.Error(err)
		}

		err = assets.GarbageCollect(state.Context, reviews)

		if err != nil {
			state.Logger.Error(err)
		}
	}()

	return api.DefaultResponse(http.StatusNoContent)
}
