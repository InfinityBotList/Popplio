package upload_asset

import (
	"bytes"
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

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"golang.org/x/image/webp"
)

const maxAssetSize = 10 * 1024 * 1024
const maxX = 1024
const maxY = 256

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
	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")

	if uid == "" || targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
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
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has(targetType, teams.PermissionAssets) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to manage assets for this entity"},
		}
	}

	// Read payload from body
	var payload types.Asset

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if payload.Type == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Type must be specified"},
		}
	}

	if len(payload.Content) == 0 || len(payload.Content) > maxAssetSize {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Content must be between 1 and 10mb"},
		}
	}

	switch payload.Type {
	case "banner":
		if payload.ContentType == "" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "ContentType must be specified to upload a banner"},
			}
		}

		reader := bytes.NewReader(payload.Content)

		var img image.Image
		var fileExt string
		switch payload.ContentType {
		case "image/png":
			fileExt = "png"

			// decode image
			img, err = png.Decode(reader)

			if err != nil {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Error decoding PNG: " + err.Error()},
				}
			}
		case "image/jpeg":
			fileExt = "jpg"

			// decode image
			img, err = jpeg.Decode(reader)

			if err != nil {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Error decoding JPEG: " + err.Error()},
				}
			}
		case "image/gif":
			fileExt = "gif"

			// decode image
			img, err = gif.Decode(reader)

			if err != nil {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Error decoding GIF: " + err.Error()},
				}
			}
		case "image/webp":
			fileExt = "webp"

			// decode image
			img, err = webp.Decode(reader)

			if err != nil {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Error decoding WEBP: " + err.Error()},
				}
			}
		default:
			return uapi.HttpResponse{
				Status: http.StatusNotImplemented,
				Json:   types.ApiError{Message: "ContentType not implemented for this banner"},
			}
		}

		// check image size
		if (maxX != 0 && maxY != 0) && (img.Bounds().Dx() > maxX || img.Bounds().Dy() > maxY) {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Image must be 1024x256 or smaller"},
			}
		}

		// Save image to temp file
		filePath := os.TempDir() + "pconv_" + crypto.RandString(256) + "." + fileExt
		targetPath := state.Config.Meta.CDNPath + "/banners/" + targetType + "/" + targetId + ".webp"
		tmpfile, err := os.Create(filePath)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Error creating temp file: " + err.Error()},
			}
		}

		defer func() {
			err := tmpfile.Close()
			if err != nil {
				state.Logger.Error(err)
			}

			err = os.Remove(filePath)

			if err != nil {
				state.Logger.Error(err)
			}
		}()

		// Convert to webp
		cmd := []string{"cwebp", "-q", "100", filePath, "-o", targetPath, "-v"}

		if fileExt == "gif" {
			// use gif2webp instead
			cmd = []string{"gif2webp", "-q", "100", "-m", "3", filePath, "-o", targetPath, "-v"}
		}

		outbuf := bytes.NewBuffer(nil)

		cmdExec := exec.Command(cmd[0], cmd[1:]...)
		cmdExec.Stdout = outbuf
		cmdExec.Stderr = outbuf
		cmdExec.Env = os.Environ()

		err = cmdExec.Run()

		outputCmd := outbuf.String()

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Error converting image: " + err.Error() + "\n" + outputCmd},
			}
		}

		return uapi.DefaultResponse(http.StatusNoContent)
	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "Asset type not implemented"},
		}
	}
}
