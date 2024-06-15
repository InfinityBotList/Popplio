package upload_asset

import (
	"fmt"
	"net/http"
	"popplio/assetmanager"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"time"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Upload Asset",
		Description: "Uploads an asset for entity to the server with a max-file limit of 10mb. Note that the user must have 'Upload Assets' permissions on the entity. Returns 204 on success",
		Req:         types.Asset{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the tntity",
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
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if uid == "" || targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Both target_id and target_type must be specified"},
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
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("target_type", targetType), zap.String("target_id", targetId))
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: targetType, Perm: teams.PermissionUploadAssets}) {
		return uapi.HttpResponse{
			Status:  http.StatusForbidden,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "You do not have permission to upload assets for this entity"},
		}
	}

	// Read payload from body
	var payload types.Asset

	hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	if payload.Type == "" {
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Type must be specified"},
		}
	}

	if len(payload.Content) == 0 || len(payload.Content) > assetmanager.MaxAssetSize {
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Content must be between 1 and 10mb"},
		}
	}

	switch payload.Type {
	case "banner":
		if payload.ContentType == "" {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: "ContentType must be specified to upload a banner"},
			}
		}

		fileExt, img, err := assetmanager.DecodeImage(&payload)

		if err != nil {
			state.Logger.Error("Error decoding image", zap.Error(err), zap.String("content_type", payload.ContentType), zap.String("type", payload.Type), zap.String("target_type", targetType), zap.String("target_id", targetId))
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: err.Error()},
			}
		}

		// check image size
		if assetmanager.BannerMaxX != 0 && assetmanager.BannerMaxY != 0 {
			if img.Bounds().Dx() > assetmanager.BannerMaxX || img.Bounds().Dy() > assetmanager.BannerMaxY {
				return uapi.HttpResponse{
					Status:  http.StatusBadRequest,
					Headers: limit.Headers(),
					Json:    types.ApiError{Message: fmt.Sprintf("Image must be %dx%d or smaller", assetmanager.BannerMaxX, assetmanager.BannerMaxY)},
				}
			}
		}

		// Save image to temp file
		err = assetmanager.EncodeImageToFile(
			img,
			func() string {
				if fileExt == "gif" {
					return "gif"
				}

				return "jpg"
			}(),
			state.Config.Meta.CDNPath+"/banners/"+targetType+"s/"+targetId+".webp",
		)

		if err != nil {
			return uapi.HttpResponse{
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: "Error converting image: " + err.Error()},
			}
		}

		return uapi.HttpResponse{
			Status:  http.StatusNoContent,
			Headers: limit.Headers(),
		}
	case "avatar":
		if targetType == "bot" {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: "Cannot upload an avatar for a bot"},
			}
		}

		if payload.ContentType == "" {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: "ContentType must be specified to upload an avatar"},
			}
		}

		fileExt, img, err := assetmanager.DecodeImage(&payload)

		if err != nil {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: err.Error()},
			}
		}

		// check image size
		if assetmanager.AvatarMaxX != 0 && assetmanager.AvatarMaxY != 0 {
			if img.Bounds().Dx() > assetmanager.AvatarMaxX || img.Bounds().Dy() > assetmanager.AvatarMaxY {
				return uapi.HttpResponse{
					Status:  http.StatusBadRequest,
					Headers: limit.Headers(),
					Json:    types.ApiError{Message: fmt.Sprintf("Image must be %dx%d or smaller", assetmanager.AvatarMaxX, assetmanager.AvatarMaxY)},
				}
			}
		}

		// Save image to temp file
		err = assetmanager.EncodeImageToFile(
			img,
			func() string {
				if fileExt == "gif" {
					return "gif"
				}

				return "jpg"
			}(),
			state.Config.Meta.CDNPath+"/avatars/"+targetType+"s/"+targetId+".webp",
		)

		if err != nil {
			return uapi.HttpResponse{
				Status:  http.StatusInternalServerError,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: "Error converting image: " + err.Error()},
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
