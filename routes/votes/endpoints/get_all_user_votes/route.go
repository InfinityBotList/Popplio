// Package get_all_user_votes implements GET
// /users/{uid}/{target_type}/{target_id}/votes/@all — "Get All User Votes".
//
// Gets all votes (paginated by 10) of a user on an entity. This endpoint is
// currently public as the same data can be found through #vote-logs in
// discord. Note that for compatibility, a trailing 's' is removed
package get_all_user_votes

import (
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/pagination"
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

	if uid == "" || targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+entityVoteCols+" FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND author = $3 LIMIT $4 OFFSET $5", targetId, targetType, uid, limit, offset)

	if err != nil {
		return resp.Err("Failed to get user entity votes", err, zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	ev, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.EntityVote])

	if err != nil {
		return resp.Err("Failed to get user entity votes", err, zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND author = $3", targetId, targetType, uid).Scan(&count)

	if err != nil {
		return resp.Err("Failed to get user entity votes", err, zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
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
