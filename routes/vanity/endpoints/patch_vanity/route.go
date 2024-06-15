package patch_vanity

import (
	"net/http"
	"slices"
	"strings"
	"unicode"

	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-playground/validator/v10"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var compiledMessages = uapi.CompileValidationErrors(types.PatchVanity{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Entity Vanity",
		Description: "Updates an entities vanity. Returns 204 on success",
		Req:         types.PatchVanity{},
		Params: []docs.Parameter{
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
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id, target_type must be specified"},
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

	// Read payload from body
	var payload types.PatchVanity

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

	if payload.Code == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Vanity cannot be empty"},
		}
	}

	// Strip out unicode characters and validate vanity
	vanity := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, payload.Code)

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
