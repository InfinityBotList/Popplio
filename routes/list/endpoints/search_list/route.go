package search_list

import (
	_ "embed"
	"net/http"
	"strings"
	"text/template"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-playground/validator/v10"
)

var (
	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	//go:embed sql/bots.tmpl
	botsSql string

	botSqlTemplate *template.Template

	compiledMessages = uapi.CompileValidationErrors(types.SearchQuery{})
)

type searchSqlTemplateCtx struct {
	Query          string
	TagMode        types.TagMode
	Cols           string
	PlatformTables []string
}

func Setup() {
	botSqlTemplate = template.Must(template.New("sql").Parse(botsSql))
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Search List",
		Description: "Searches the list. This replaces arcadias tetanus API",
		Req:         types.SearchQuery{},
		Resp:        types.SearchResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.SearchQuery

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if payload.Query == "" && len(payload.TagFilter.Tags) == 0 {
		return uapi.HttpResponse{
			Json: types.SearchResponse{
				Bots: []types.IndexBot{},
			},
		}
	}

	// Default, if not specified
	if payload.TagFilter.TagMode == "" {
		payload.TagFilter.TagMode = types.TagModeAny
	}

	if len(payload.TagFilter.Tags) == 0 {
		payload.TagFilter.Tags = []string{}
	}

	if payload.TagFilter.TagMode != types.TagModeAll && payload.TagFilter.TagMode != types.TagModeAny {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Invalid tag mode"},
		}
	}

	var indexBots = []types.IndexBot{}

	sqlString := &strings.Builder{}

	err = botSqlTemplate.Execute(sqlString, searchSqlTemplateCtx{
		Query:   payload.Query,
		TagMode: payload.TagFilter.TagMode,
		Cols:    indexBotCols,
		PlatformTables: []string{
			dovewing.TableName(state.DovewingPlatformDiscord),
		},
	})

	state.Logger.Error(sqlString.String())

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
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
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&indexBots, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i, bot := range indexBots {
		botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.DovewingPlatformDiscord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		indexBots[i].User = botUser
	}

	return uapi.HttpResponse{
		Json: types.SearchResponse{
			Bots: indexBots,
		},
	}
}
