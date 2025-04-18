package get_all_user_votes

import (
	"net/http"
	"strconv"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const perPage = 5

var (
	entityVoteColsArr = db.GetCols(types.EntityVote{})
	entityVoteCols    = strings.Join(entityVoteColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All User Votes",
		Description: "Gets all votes (paginated by 10) of a user on an entity. This endpoint is currently public as the same data can be found through #vote-logs in discord. Note that for compatibility, a trailing 's' is removed",
		Resp:        types.PagedResult[[]types.EntityVote]{},
		RespName:    "PagedResultUserVote",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
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
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	if uid == "" || targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+entityVoteCols+" FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND author = $3 LIMIT $4 OFFSET $5", targetId, targetType, uid, limit, offset)

	if err != nil {
		state.Logger.Error("Failed to get user entity votes", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	ev, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.EntityVote])

	if err != nil {
		state.Logger.Error("Failed to get user entity votes", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND author = $3", targetId, targetType, uid).Scan(&count)

	if err != nil {
		state.Logger.Error("Failed to get user entity votes", zap.Error(err), zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[[]types.EntityVote]{
		Count:   count,
		PerPage: perPage,
		Results: ev,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
