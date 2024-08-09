package create_session

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateSession{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Session",
		Description: "Creates a new session returning the session token. The session token cannot be read after creation.",
		Req:         types.CreateSession{},
		Resp:        types.CreateSessionResponse{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The entity type to use",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID to use",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Missing target_id or target_type"},
		}
	}

	targetType = strings.TrimSuffix(targetType, "s")

	var createData types.CreateSession

	hresp, ok := uapi.MarshalReq(r, &createData)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(createData)

	if err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	if createData.Name == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Name is required"},
		}
	}

	if createData.Type == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Type is required"},
		}
	}

	if createData.Expiry <= 0 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Expiry must be greater than or equal to zero"},
		}
	}

	if len(createData.PermLimits) == 0 {
		createData.PermLimits = []string{}
	}

	// The outer perm limit stores what permissions the session is limited to in creation
	//
	// SAFETY: A 'user' has to initially start the process of creating a session
	//
	// This means that we only need to check entity perms for users, then the checks occur based on
	// the outer perm limits
	var outerPermLimit []perms.Permission
	switch d.Auth.TargetType {
	case api.TargetTypeUser:
		// Get user entity perms
		managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

		if err != nil {
			state.Logger.Error("Error while getting entity perms", zap.Error(err))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Error while getting entity perms: " + err.Error()},
			}
		}

		if len(managerPerms) == 0 {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "User does not have any permissions on this eneity whatsoever!"},
			}
		}

		// Set outer perm limit to the manager perms, see safety note above
		outerPermLimit = managerPerms
	default:
		pl := api.PermLimits(d.Auth)

		if len(pl) > 0 {
			outerPermLimit = perms.PFSS(pl)
		}
	}

	if len(outerPermLimit) == 0 {
		// Assume no permission limits if no outer perm limits are set
		outerPermLimit = []perms.Permission{
			{
				Namespace: "global",
				Perm:      teams.PermissionOwner,
			},
		}
	}

	// All permission limits must be resolved before being added to db
	permLimits := perms.StaffPermissions{
		PermOverrides: perms.PFSS(createData.PermLimits),
	}.Resolve()

	if !perms.HasPerm(outerPermLimit, perms.Permission{Namespace: "global", Perm: teams.PermissionOwner}) {
		if len(createData.PermLimits) == 0 {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You must have Global Owner to create sessions without specifying a permission limit"},
			}
		}

		for _, perm := range permLimits {
			if !perms.HasPerm(outerPermLimit, perm) {
				return uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "User does not have permission to create sessions with the permission limit: " + perm.String()},
				}
			}
		}
	}

	// Create session
	sessionToken := crypto.RandString(128)
	var sessionId string

	expiry := time.Now().Add(time.Duration(createData.Expiry) * time.Second)

	err = state.Pool.QueryRow(
		d.Context,
		"INSERT INTO api_sessions (token, target_id, target_type, name, type, expiry, perm_limits) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		sessionToken,
		targetId,
		targetType,
		createData.Name,
		createData.Type,
		expiry,
		permLimits,
	).Scan(&sessionId)

	if err != nil {
		state.Logger.Error("Error while creating user session", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error while creating user session: " + err.Error()},
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: types.CreateSessionResponse{
			TargetID:  targetId,
			Token:     sessionToken,
			SessionID: sessionId,
		},
	}

}
