package bp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"popplio/assetmanager"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"go.uber.org/zap"
)

func updateAvatarCache(ctx context.Context, typ string, id string, avatarUrl string) error {
	// Download avatar from url
	c := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", avatarUrl, nil)

	if err != nil {
		return err
	}

	resp, err := c.Do(req)

	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		return nil // Discord moment
	}

	var contentType = resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	defer resp.Body.Close()

	// Save avatar to cache
	avatarBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payload := types.Asset{Type: "avatar", ContentType: contentType, Content: avatarBytes}

	fileExt, img, err := assetmanager.DecodeImage(&payload)

	if err != nil {
		return errors.New("error decoding image: " + err.Error())
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err = assetmanager.EncodeImageToFile(
		img,
		func() string {
			if fileExt == "gif" {
				return "gif"
			}

			return "jpg"
		}(),
		state.Config.Meta.CDNPath+"/avatars/"+typ+"/"+id+".webp",
	)

	if err != nil {
		return errors.New("error encoding image to temp file: " + err.Error())
	}

	return nil
}

func DovewingMiddleware(p dovewing.Platform, pu *dovetypes.PlatformUser) (*dovetypes.PlatformUser, error) {
	var typ = assetmanager.AssetTargetTypeBots

	if !pu.Bot {
		typ = assetmanager.AssetTargetTypeUsers
	}

	avatar := assetmanager.AvatarInfo(typ, pu.ID)

	if !avatar.Exists || time.Since(*avatar.LastModified) > time.Hour*8 {
		state.Logger.Info("Updating avatar cache", zap.String("id", pu.ID))

		err := updateAvatarCache(state.Context, typ.String(), pu.ID, pu.Avatar)

		if err != nil {
			return pu, errors.New("error updating avatar cache: " + err.Error())
		}

		avatar = assetmanager.AvatarInfo(typ, pu.ID)
	}

	if len(pu.ExtraData) == 0 {
		pu.ExtraData = make(map[string]interface{})
	}

	pu.ExtraData["avatar"] = avatar

	pu.Avatar = assetmanager.ResolveAssetMetadataToUrl(avatar)

	return pu, nil
}
