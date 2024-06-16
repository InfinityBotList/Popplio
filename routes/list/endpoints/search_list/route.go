package search_list

import (
	_ "embed"
	"net/http"
	"strings"
	"text/template"

	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	serverAssets "popplio/routes/servers/assets"
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
	indexBotColsArr           = db.GetCols(types.IndexBot{})
	indexBotColsWithPrefixArr = func() []string {
		// Prefix all columns with bots.
		var cols []string

		for _, col := range indexBotColsArr {
			cols = append(cols, "bots."+col)
		}

		return cols
	}()

	indexBotColsWithPrefix = strings.Join(indexBotColsWithPrefixArr, ",")

	indexServerColsArr = db.GetCols(types.IndexServer{})
	indexServerCols    = strings.Join(indexServerColsArr, ",")

	compiledMessages = uapi.CompileValidationErrors(types.SearchQuery{})
)

var (
	//go:embed sql/bots.tmpl
	botsSql        string
	botSqlTemplate *template.Template

	//go:embed sql/servers.tmpl
	serversSql        string
	serverSqlTemplate *template.Template
)

type searchSqlTemplateCtx struct {
	Query          string
	TagMode        types.TagMode
	Cols           string
	PlatformTables []string
}

func Setup() {
	botSqlTemplate = template.Must(template.New("sqlA").Parse(botsSql))
	serverSqlTemplate = template.Must(template.New("sqlB").Parse(serversSql))
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Search List",
		Description: "Searches the list returning a list of bots/servers that match the query",
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
		// Return 206 because the user didn't specify a query or tags
		//
		// Clients can then use this to not show any bots
		return uapi.HttpResponse{
			Status: http.StatusPartialContent,
			Json:   types.ApiError{Message: "No query or tags specified"},
		}
	}

	// Default, if not specified
	if payload.TagFilter.TagMode == "" {
		payload.TagFilter.TagMode = types.TagModeAny
	}

	if len(payload.TagFilter.Tags) == 0 {
		payload.TagFilter.Tags = []string{}
	}

	if len(payload.TargetTypes) == 0 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "No target types specified"},
		}
	}

	if payload.TagFilter.TagMode != types.TagModeAll && payload.TagFilter.TagMode != types.TagModeAny {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Invalid tag mode"},
		}
	}

	sr := types.SearchResponse{}

	for _, targetType := range payload.TargetTypes {
		switch targetType {
		case "bot":
			sr.TargetTypes = append(sr.TargetTypes, "bot")
			sqlString := &strings.Builder{}

			err = botSqlTemplate.Execute(sqlString, searchSqlTemplateCtx{
				Query:   payload.Query,
				TagMode: payload.TagFilter.TagMode,
				Cols:    indexBotColsWithPrefix, // We need to prefix the columns with bots. to avoid ambiguity
				PlatformTables: []string{
					dovewing.TableName(state.DovewingPlatformDiscord),
				},
			})

			if err != nil {
				state.Logger.Error("Failed to execute template", zap.Error(err), zap.String("sql", sqlString.String()))
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
				args = append(args, strings.ToLower(payload.Query), "%"+strings.ToLower(payload.Query)+"%") // 8-9
			}

			state.Logger.Debug("SQL result", zap.String("sql", sqlString.String()), zap.String("targetType", "bot"))

			rows, err := state.Pool.Query(
				d.Context,
				sqlString.String(),
				// Args
				args...,
			)

			if err != nil {
				state.Logger.Error("Failed to query", zap.Error(err), zap.String("targetType", "bot"))
				return uapi.HttpResponse{
					Status: http.StatusInternalServerError,
					Json:   types.ApiError{Message: "Error querying: " + err.Error()},
				}
			}

			bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

			if err != nil {
				state.Logger.Error("Failed to collect rows [bots]", zap.Error(err), zap.String("sql", sqlString.String()))
				return uapi.HttpResponse{
					Status: http.StatusInternalServerError,
					Json:   types.ApiError{Message: "Error collecting rows: " + err.Error()},
				}
			}

			for i := range bots {
				err := botAssets.ResolveIndexBot(d.Context, &bots[i])

				if err != nil {
					return uapi.HttpResponse{
						Status: http.StatusInternalServerError,
						Json:   types.ApiError{Message: "Error resolving bot: " + err.Error()},
					}
				}
			}

			sr.Bots = bots
		case "server":
			sr.TargetTypes = append(sr.TargetTypes, "server")

			sqlString := &strings.Builder{}

			err = serverSqlTemplate.Execute(sqlString, searchSqlTemplateCtx{
				Query:   payload.Query,
				TagMode: payload.TagFilter.TagMode,
				Cols:    indexServerCols,
			})

			if err != nil {
				state.Logger.Error("Failed to execute template", zap.Error(err), zap.String("sql", sqlString.String()))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			args := []any{
				payload.TotalMembers.From, // 1
				payload.TotalMembers.To,   // 2
				payload.Votes.From,        // 3
				payload.Votes.To,          // 4
				payload.TagFilter.Tags,    // 5
			}

			if payload.Query != "" {
				args = append(args, "%"+strings.ToLower(payload.Query)+"%", strings.ToLower(payload.Query)) // 6-7
			}

			state.Logger.Debug("SQL result", zap.String("sql", sqlString.String()), zap.String("targetType", "server"))

			rows, err := state.Pool.Query(
				d.Context,
				sqlString.String(),
				// Args
				args...,
			)

			if err != nil {
				state.Logger.Error("Failed to query", zap.Error(err), zap.String("targetType", "server"))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

			if err != nil {
				state.Logger.Error("Failed to collect rows", zap.Error(err), zap.String("sql", sqlString.String()))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			for i := range servers {
				err := serverAssets.ResolveIndexServer(d.Context, &servers[i])

				if err != nil {
					state.Logger.Error("Failed to resolve server", zap.Error(err), zap.String("serverId", servers[i].ServerID))
					return uapi.HttpResponse{
						Status: http.StatusInternalServerError,
						Json:   types.ApiError{Message: "Error resolving server: " + err.Error()},
					}
				}
			}

			sr.Servers = servers
		}
	}

	if len(sr.TargetTypes) == 0 {
		sr.TargetTypes = []string{}
	}

	return uapi.HttpResponse{
		Json: sr,
	}
}
