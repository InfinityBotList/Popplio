package delete_asset

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"time"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/infinitybotlist/eureka/uapi/ratelimit"
	"go.uber.org/zap"
)

// Deletes a file if it exists
func deleteFileIfExists(path string) error {
	st, err := os.Stat(path)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	}

	if st.IsDir() {
		return errors.New("path is a directory")
	}

	// Delete file
	err = os.Remove(path)

	if err != nil {
		return err
	}

	return nil
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Asset",
		Description: "Deletes an asset for an entity. User must have 'Delete Assets' permissions on the entity. Returns 204 on success",
		Resp:        types.ApiError{},
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
				Name:        "type",
				Description: "The type of asset to delete.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 3,
		Bucket:      "assets",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error("Error while ratelimiting", zap.Error(err), zap.String("bucket", "assets"))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")
	assetType := r.URL.Query().Get("type")

	if uid == "" || targetId == "" || targetType == "" || assetType == "" {
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "uid, target_id, target_type and type must be specified"},
		}
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return uapi.HttpResponse{
			Status:  http.StatusNotImplemented,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Target type not implemented"},
		}
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err))
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has(targetType, teams.PermissionDeleteAssets) {
		return uapi.HttpResponse{
			Status:  http.StatusForbidden,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "You do not have permission to delete assets for this entity"},
		}
	}

	switch assetType {
	case "banner":
		err = deleteFileIfExists(state.Config.Meta.CDNPath + "/banners/" + targetType + "s/" + targetId + ".webp")

		if err != nil {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: err.Error()},
			}
		}

		return uapi.HttpResponse{
			Status:  http.StatusNoContent,
			Headers: limit.Headers(),
		}
	case "avatar":
		err = deleteFileIfExists(state.Config.Meta.CDNPath + "/avatars/" + targetType + "s/" + targetId + ".webp")

		if err != nil {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: err.Error()},
			}
		}

		return uapi.HttpResponse{
			Status:  http.StatusNoContent,
			Headers: limit.Headers(),
		}
	default:
		return uapi.HttpResponse{
			Status:  http.StatusNotImplemented,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Asset type not implemented"},
		}
	}
}
