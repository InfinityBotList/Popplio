package search_list

import (
	_ "embed"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-playground/validator/v10"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	//go:embed sql/bots.sql
	botsSql string
)

type SearchFilter struct {
	From int `json:"from" validate:"required"`
	To   int `json:"to" validate:"required"`
}

func (f SearchFilter) from() int {
	if f.From == 0 {
		return -1
	}
	return f.From
}

func (f SearchFilter) to() int {
	if f.To == 0 {
		return -1
	}
	return f.To
}

type TagMode string

const (
	TagModeAll TagMode = "@>"
	TagModeAny TagMode = "&&"
)

type TagFilter struct {
	Tags    []string `json:"tags" validate:"required"`
	TagMode TagMode  `json:"tag_mode" validate:"required"`
}

type SearchQuery struct {
	Query     string        `json:"query" validate:"required"`
	Servers   *SearchFilter `json:"servers" validate:"required"`
	Votes     *SearchFilter `json:"votes" validate:"required"`
	Shards    *SearchFilter `json:"shards" validate:"required"`
	TagFilter *TagFilter    `json:"tags"` // Optional for now, as main frontend doesn't support it yet
}

// Only bots are supported at this time
type SearchResponse struct {
	Bots []types.IndexBot `json:"bots"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/list/search",
		OpId:        "search_list",
		Summary:     "Search List",
		Description: "Searches the list. This replaces arcadias tetanus API",
		Tags:        []string{api.CurrentTag},
		Req:         SearchQuery{},
		Resp:        SearchResponse{},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload SearchQuery

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(api.BlankMap, errors)
	}

	if payload.TagFilter == nil {
		payload.TagFilter = &TagFilter{
			Tags:    []string{},
			TagMode: TagModeAll,
		}
	}

	if payload.TagFilter.TagMode != TagModeAll && payload.TagFilter.TagMode != TagModeAny {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid tag mode",
				Error:   true,
			},
		}
	}

	var indexBots = []types.IndexBot{}

	botsSql = strings.Replace(botsSql, "{cols}", indexBotCols, 1)
	botsSql = strings.Replace(botsSql, "{op}", string(payload.TagFilter.TagMode), 1)

	rows, err := state.Pool.Query(
		d.Context,
		botsSql,
		// Args
		payload.Query,          // 1
		"%"+payload.Query+"%",  // 2
		payload.Servers.from(), // 3
		payload.Servers.to(),   // 4
		payload.Votes.from(),   // 5
		payload.Votes.to(),     // 6
		payload.Shards.from(),  // 7
		payload.Shards.to(),    // 8
		payload.TagFilter.Tags, // 9
	)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&indexBots, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	indexBots, err = utils.ResolveIndexBot(indexBots)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: SearchResponse{
			Bots: indexBots,
		},
	}
}
