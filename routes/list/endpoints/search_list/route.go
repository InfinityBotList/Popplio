package search_list

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	//go:embed sql/bots.sql
	botsSql string
)

type SearchFilter struct {
	From int `json:"from"`
	To   int `json:"to"`
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
	Tags    []string `json:"tags"`
	TagMode TagMode  `json:"tag_mode"`
}

type SearchQuery struct {
	Query     string        `json:"query"`
	Servers   *SearchFilter `json:"servers"`
	Votes     *SearchFilter `json:"votes"`
	Shards    *SearchFilter `json:"shards"`
	TagFilter *TagFilter    `json:"tags"`
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

func Route(d api.RouteData, r *http.Request) {
	defer r.Body.Close()

	var payload SearchQuery

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if len(bodyBytes) == 0 {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "A body is required for this endpoint",
				Error:   true,
			},
		}
		return
	}

	err = json.Unmarshal(bodyBytes, &payload)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid JSON:" + err.Error(),
				Error:   true,
			},
		}
		return
	}

	if payload.Servers == nil {
		payload.Servers = &SearchFilter{
			From: -1,
			To:   -1,
		}
	}

	if payload.Votes == nil {
		payload.Votes = &SearchFilter{
			From: -1,
			To:   -1,
		}
	}

	if payload.Shards == nil {
		payload.Shards = &SearchFilter{
			From: -1,
			To:   -1,
		}
	}

	if payload.TagFilter == nil {
		payload.TagFilter = &TagFilter{
			Tags:    []string{},
			TagMode: TagModeAll,
		}
	}

	if payload.TagFilter.TagMode != TagModeAll && payload.TagFilter.TagMode != TagModeAny {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid tag mode",
				Error:   true,
			},
		}
		return
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
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	err = pgxscan.ScanAll(&indexBots, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	indexBots, err = utils.ResolveIndexBot(indexBots)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	d.Resp <- api.HttpResponse{
		Json: SearchResponse{
			Bots: indexBots,
		},
	}
}
