package patch_vanity

import (
	"net/http"
	"slices"
	"strings"
	"unicode"

	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Entity Vanity",
		Description: "Updates an entities vanity. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the tntity",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "vanity",
				Description: "The new vanity",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")
	vanity := r.URL.Query().Get("vanity")

	if uid == "" || targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	if vanity == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot be empty"},
		}
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "Target type not implemented"},
		}
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: targetType, Perm: teams.PermissionSetVanity}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to update this entities vanity"},
		}
	}

	// Strip out unicode characters and validate vanity
	vanity = strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, vanity)

	systems, err := validators.GetWordBlacklistSystems(d.Context, vanity)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error while getting word blacklist systems: " + err.Error()},
		}

	}

	if slices.Contains(systems, "vanity.code") {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "The chosen vanity is blacklisted"},
		}
	}

	if strings.Contains(vanity, "@") {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot contain @"},
		}
	}

	vanity = strings.TrimSuffix(vanity, "-")
	vanity = strings.ToLower(vanity)
	vanity = strings.ReplaceAll(vanity, " ", "-")

	if vanity == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot be empty"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Ensure vanity doesn't already exist
	var count int64

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE code = $1", vanity).Scan(&count)

	if err != nil {
		state.Logger.Error("Error while querying vanity", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count > 0 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity is already taken"},
		}
	}

	// Check that a vanity row exists
	var rowCount int64
	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&rowCount)

	if err != nil {
		state.Logger.Error("Error while querying vanity", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if rowCount == 0 {
		_, err = tx.Exec(d.Context, "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, $2, $3)", targetId, targetType, vanity)

		if err != nil {
			state.Logger.Error("Error while inserting vanity", zap.Error(err), zap.String("userID", d.Auth.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	} else {
		// Update vanity
		_, err = tx.Exec(d.Context, "UPDATE vanity SET code = $1 WHERE target_id = $2 AND target_type = $3", vanity, targetId, targetType)

		if err != nil {
			state.Logger.Error("Error while updating vanity", zap.Error(err), zap.String("userID", d.Auth.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
