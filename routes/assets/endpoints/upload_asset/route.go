package upload_asset

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"
	"golang.org/x/image/webp"
)

const maxAssetSize = 50 * 1024 * 1024
const bannerMaxX = 0 // STILL DECIDING
const bannerMaxY = 0 // STILL DECIDING
const avatarMaxX = 0 // STILL DECIDING
const avatarMaxY = 0 // STILL DECIDING

func decodeImage(payload *types.Asset) (fileExt string, img image.Image, err error) {
	reader := bytes.NewReader(payload.Content)

	switch payload.ContentType {
	case "image/png":
		// decode image
		img, err = png.Decode(reader)

		if err != nil {
			return "png", nil, fmt.Errorf("error decoding PNG: %s", err.Error())
		}

		return "png", img, nil
	case "image/jpeg":
		// decode image
		img, err = jpeg.Decode(reader)

		if err != nil {
			return "jpg", nil, fmt.Errorf("error decoding JPEG: %s", err.Error())
		}

		return "jpg", img, nil
	case "image/gif":
		// decode image
		img, err = gif.Decode(reader)

		if err != nil {
			return "gif", nil, fmt.Errorf("error decoding GIF: %s", err.Error())
		}

		return "gif", img, nil
	case "image/webp":
		// decode image
		img, err = webp.Decode(reader)

		if err != nil {
			return "webp", nil, fmt.Errorf("error decoding GIF: %s", err.Error())
		}

		return "webp", img, nil
	default:
		return "", nil, fmt.Errorf("content type not implemented")
	}
}

func encodeImageToFile(img image.Image, intermediary, outputPath string) error {
	var tmpPath = os.TempDir() + "/pconv_" + crypto.RandString(128) + "." + intermediary

	tmpfile, err := os.Create(tmpPath)

	if err != nil {
		return fmt.Errorf("error creating temp file: %s", err.Error())
	}

	if intermediary == "gif" {
		err = gif.Encode(tmpfile, img, &gif.Options{})

		if err != nil {
			return fmt.Errorf("error encoding image to temp file: %s", err.Error())
		}
	} else {
		err = jpeg.Encode(tmpfile, img, &jpeg.Options{Quality: 100})

		if err != nil {
			return fmt.Errorf("error encoding image to temp file: %s", err.Error())
		}
	}

	err = tmpfile.Close()

	if err != nil {
		return fmt.Errorf("error closing temp file: %s", err.Error())
	}

	cmd := []string{"cwebp", "-q", "100", tmpPath, "-o", outputPath, "-v"}

	if intermediary == "gif" {
		cmd = []string{"gif2webp", "-q", "100", "-m", "3", tmpPath, "-o", outputPath, "-v"}
	}

	outbuf := bytes.NewBuffer(nil)

	cmdExec := exec.Command(cmd[0], cmd[1:]...)
	cmdExec.Stdout = outbuf
	cmdExec.Stderr = outbuf
	cmdExec.Env = os.Environ()

	err = cmdExec.Run()

	outputCmd := outbuf.String()

	if err != nil {
		return fmt.Errorf("error converting image: %s\n%s", err.Error(), outputCmd)
	}

	// Delete temp file
	err = os.Remove(tmpPath)

	if err != nil {
		return fmt.Errorf("error deleting temp file: %s", err.Error())
	}

	return nil
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Upload Asset",
		Description: "Uploads an asset for entity to the server with a max-file limit of 10mb. Note that the user must have 'Upload Assets' permissions on the entity. Returns 204 on success",
		Req:         types.Asset{},
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

	if !kittycat.HasPerm(perms, kittycat.Build(targetType, teams.PermissionUploadAssets)) {
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

	if len(payload.Content) == 0 || len(payload.Content) > maxAssetSize {
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

		fileExt, img, err := decodeImage(&payload)

		if err != nil {
			state.Logger.Error("Error decoding image", zap.Error(err), zap.String("content_type", payload.ContentType), zap.String("type", payload.Type), zap.String("target_type", targetType), zap.String("target_id", targetId))
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: err.Error()},
			}
		}

		// check image size
		if bannerMaxX != 0 && bannerMaxY != 0 {
			if img.Bounds().Dx() > bannerMaxX || img.Bounds().Dy() > bannerMaxY {
				return uapi.HttpResponse{
					Status:  http.StatusBadRequest,
					Headers: limit.Headers(),
					Json:    types.ApiError{Message: fmt.Sprintf("Image must be %dx%d or smaller", bannerMaxX, bannerMaxY)},
				}
			}
		}

		// Save image to temp file
		err = encodeImageToFile(
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
		if payload.ContentType == "" {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: "ContentType must be specified to upload an avatar"},
			}
		}

		fileExt, img, err := decodeImage(&payload)

		if err != nil {
			return uapi.HttpResponse{
				Status:  http.StatusBadRequest,
				Headers: limit.Headers(),
				Json:    types.ApiError{Message: err.Error()},
			}
		}

		// check image size
		if avatarMaxX != 0 && avatarMaxY != 0 {
			if img.Bounds().Dx() > avatarMaxX || img.Bounds().Dy() > avatarMaxY {
				return uapi.HttpResponse{
					Status:  http.StatusBadRequest,
					Headers: limit.Headers(),
					Json:    types.ApiError{Message: fmt.Sprintf("Image must be %dx%d or smaller", avatarMaxX, avatarMaxY)},
				}
			}
		}

		// Save image to temp file
		err = encodeImageToFile(
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
