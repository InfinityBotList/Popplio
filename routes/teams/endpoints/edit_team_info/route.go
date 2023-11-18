package edit_team_info

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	cevents "popplio/webhooks/core/events"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateEditTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Team Info",
		Description: "Edits a team. Returns a 204 on success.",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tid",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.CreateEditTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload types.CreateEditTeam

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Ensure manager has perms to edit this team
	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has("team", teams.PermissionEdit) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit this team's information (name/avatar/mention)"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error beginning transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Get current name and avatar
	var oldName string
	var oldShort pgtype.Text
	var oldTags []string
	var oldExtraLinks []types.Link
	var oldNsfw bool

	err = tx.QueryRow(d.Context, "SELECT name, short, tags, extra_links, nsfw FROM teams WHERE id = $1", teamId).Scan(&oldName, &oldShort, &oldTags, &oldExtraLinks, &oldNsfw)

	if err != nil {
		state.Logger.Error("Error getting team info [db queryrow]", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Update the team
	_, err = tx.Exec(d.Context, "UPDATE teams SET name = $1 WHERE id = $2", payload.Name, teamId)

	if err != nil {
		state.Logger.Error("Error updating team info", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if payload.Short != nil {
		_, err = tx.Exec(d.Context, "UPDATE teams SET short = $1 WHERE id = $2", payload.Short, teamId)

		if err != nil {
			state.Logger.Error("Error updating team info", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Tags != nil {
		_, err = tx.Exec(d.Context, "UPDATE teams SET tags = $1 WHERE id = $2", payload.Tags, teamId)

		if err != nil {
			state.Logger.Error("Error updating team info", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.ExtraLinks != nil {
		err = validators.ValidateExtraLinks(*payload.ExtraLinks)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: err.Error()},
			}
		}

		_, err = tx.Exec(d.Context, "UPDATE teams SET extra_links = $1 WHERE id = $2", payload.ExtraLinks, teamId)

		if err != nil {
			state.Logger.Error("Error updating team info", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	var isTeamNsfw = false
	if payload.NSFW != nil {
		isTeamNsfw = *payload.NSFW
	}

	if payload.Tags != nil {
		tagList := *payload.Tags

		for _, tag := range tagList {
			if cases.Lower(language.English).String(tag) == "nsfw" {
				isTeamNsfw = true
			}
		}
	}

	if isTeamNsfw != oldNsfw {
		_, err = tx.Exec(d.Context, "UPDATE teams SET nsfw = $1 WHERE id = $2", isTeamNsfw, teamId)

		if err != nil {
			state.Logger.Error("Error updating team info", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = drivers.Send(drivers.With{
		Data: events.WebhookTeamEditData{
			Name: cevents.Changeset[string]{
				Old: oldName,
				New: payload.Name,
			},
			Short: func() cevents.Changeset[string] {
				if payload.Short == nil {
					return cevents.Changeset[string]{
						Old: oldShort.String,
						New: "",
					}
				}

				return cevents.Changeset[string]{
					Old: oldShort.String,
					New: *payload.Short,
				}
			}(),
			Tags: func() cevents.Changeset[[]string] {
				if payload.Tags == nil {
					return cevents.Changeset[[]string]{}
				}

				return cevents.Changeset[[]string]{
					Old: oldTags,
					New: *payload.Tags,
				}
			}(),
			ExtraLinks: func() cevents.Changeset[[]types.Link] {
				if payload.ExtraLinks == nil {
					return cevents.Changeset[[]types.Link]{
						Old: oldExtraLinks,
						New: []types.Link{},
					}
				}

				return cevents.Changeset[[]types.Link]{
					Old: oldExtraLinks,
					New: *payload.ExtraLinks,
				}
			}(),
			NSFW: cevents.Changeset[bool]{
				Old: oldNsfw,
				New: isTeamNsfw,
			},
		},
		UserID:     d.Auth.ID,
		TargetType: "team",
		TargetID:   teamId,
	})

	if err != nil {
		state.Logger.Error("Error sending team edit webhook", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
