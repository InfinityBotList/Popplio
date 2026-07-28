package assetmanager

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"popplio/types"

	"github.com/infinitybotlist/eureka/crypto"
	"golang.org/x/image/webp"
)

const MaxAssetSize = 50 * 1024 * 1024
const BannerMaxX = 0 // STILL DECIDING
const BannerMaxY = 0 // STILL DECIDING
const AvatarMaxX = 0 // STILL DECIDING
const AvatarMaxY = 0 // STILL DECIDING

func DecodeImage(payload *types.Asset) (fileExt string, img image.Image, err error) {
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

func EncodeImageToFile(img image.Image, intermediary, outputPath string) error {
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
