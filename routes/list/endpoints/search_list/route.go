package search_list

import (
	_ "embed"
	"net/http"
	"strings"
	"text/template"

	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-playground/validator/v10"
)

var (
	indexBotColsArr = db.GetCols(types.IndexBot{})
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

	sqlString := &strings.Builder{}

	err = botSqlTemplate.Execute(sqlString, searchSqlTemplateCtx{
		Query:   payload.Query,
		TagMode: payload.TagFilter.TagMode,
		Cols:    indexBotCols,
		PlatformTables: []string{
			dovewing.TableName(state.DovewingPlatformDiscord),
		},
	})

	state.Logger.Error("SQL Res", zap.String("sql", sqlString.String()))

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	args := []any{
		payload.Servers.From,   // 1
		payload.Servers.To,     // 2
		payload.Votes.From,     // 3
		payload.Votes.To,       // 4
		payload.Shards.From,    // 5
		payload.Shards.To,      // 6
		payload.TagFilter.Tags, // 7
	}

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

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range bots {
		botUser, err := dovewing.GetUser(d.Context, bots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].User = botUser

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", bots[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		bots[i].Vanity = code
		bots[i].Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeBots, bots[i].BotID)
	}

	return uapi.HttpResponse{
		Json: types.SearchResponse{
			Bots: bots,
		},
	}
}
