package edit_review

import (
	"net/http"
	"popplio/api"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type EditReview struct {
	Content string `db:"content" json:"content" validate:"required,min=5,max=4000" msg:"Content must be between 5 and 4000 characters"`
	Stars   int32  `db:"stars" json:"stars" validate:"required,min=1,max=5" msg:"Stars must be between 1 and 5 stars"`
}

var (
	compiledMessages = api.CompileValidationErrors(EditReview{})

	reviewColsArr = utils.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
)

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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload EditReview

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

	rid := chi.URLParam(r, "rid")

	var author string
	var botId string

	err = state.Pool.QueryRow(d.Context, "SELECT author, bot_id FROM reviews WHERE id = $1", rid).Scan(&author, &botId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if author != d.Auth.ID {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Error:   true,
				Message: "You are not the author of this review",
			},
		}
	}

	_, err = state.Pool.Exec(d.Context, "UPDATE reviews SET content = $1, stars = $2 WHERE id = $3", payload.Content, payload.Stars, rid)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Trigger a garbage collection step to remove any orphaned reviews
	go func() {
		rows, err := state.Pool.Query(state.Context, "SELECT "+reviewCols+" FROM reviews WHERE bot_id = $1 ORDER BY created_at ASC", botId)

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

	state.Redis.Del(d.Context, "rv-"+botId)

	return api.DefaultResponse(http.StatusNoContent)
}
