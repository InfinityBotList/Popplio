package search_list

import (
	_ "embed"
	"net/http"
	"strings"
	"text/template"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-playground/validator/v10"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	//go:embed sql/bots.sql
	botsSql string

	botSqlTemplate *template.Template
)

type searchSqlTemplateCtx struct {
	Query   string
	TagMode TagMode
	Cols    string
}

type SearchFilter struct {
	From uint32 `json:"from"`
	To   uint32 `json:"to"`
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
	Query     string        `json:"query"`
	Servers   *SearchFilter `json:"servers" validate:"required"`
	Votes     *SearchFilter `json:"votes" validate:"required"`
	Shards    *SearchFilter `json:"shards" validate:"required"`
	TagFilter *TagFilter    `json:"tags" validate:"required"`
}

type SearchResponse struct {
	Bots []types.IndexBot `json:"bots"`
}

func Setup() {
	botSqlTemplate = template.Must(template.New("sql").Parse(botsSql))
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "POST",
		Path:        "/list/search",
		Summary:     "Search List",
		Description: "Searches the list. This replaces arcadias tetanus API",
		Req:         SearchQuery{},
		Resp:        SearchResponse{},
	}
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

	if payload.Query == "" && len(payload.TagFilter.Tags) == 0 {
		return api.HttpResponse{
			Json: SearchResponse{
				Bots: []types.IndexBot{},
			},
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

	sqlString := &strings.Builder{}

	err = botSqlTemplate.Execute(sqlString, searchSqlTemplateCtx{
		Query:   payload.Query,
		TagMode: payload.TagFilter.TagMode,
		Cols:    indexBotCols,
	})

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	args := []any{}

	args = append(
		args,
		payload.Servers.From,   // 1
		payload.Servers.To,     // 2
		payload.Votes.From,     // 3
		payload.Votes.To,       // 4
		payload.Shards.From,    // 5
		payload.Shards.To,      // 6
		payload.TagFilter.Tags, // 7
	)

	if payload.Query != "" {
		args = append(args, "%"+strings.ToLower(payload.Query)+"%", strings.ToLower(payload.Query)) // 8-9
	}

	rows, err := state.Pool.Query(
		d.Context,
		sqlString.String(),
		// Args
		args...,
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

	for i, bot := range indexBots {
		botUser, err := utils.GetDiscordUser(bot.BotID)

		if err != nil {
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		indexBots[i].User = botUser
	}

	return api.HttpResponse{
		Json: SearchResponse{
			Bots: indexBots,
		},
	}
}
