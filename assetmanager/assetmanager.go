package assetmanager

import (
	"errors"
	"os"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"strconv"
	"time"
)

type AssetTargetType int

const (
	AssetTargetTypeUser    AssetTargetType = iota
	AssetTargetTypeBot     AssetTargetType = iota
	AssetTargetTypeServer  AssetTargetType = iota
	AssetTargetTypeTeam    AssetTargetType = iota
	AssetTargetTypePartner AssetTargetType = iota
)

func (a AssetTargetType) String() string {
	switch a {
	case AssetTargetTypeUser:
		return "user"
	case AssetTargetTypeBot:
		return "bot"
	case AssetTargetTypeServer:
		return "server"
	case AssetTargetTypeTeam:
		return "team"
	case AssetTargetTypePartner:
		return "partner"
	default:
		panic("invalid asset target type")
	}
}

// CdnString returns the folder name on the CDN
func (a AssetTargetType) CdnString() string {
	switch a {
	case AssetTargetTypeUser:
		return "users"
	case AssetTargetTypeBot:
		return "bots"
	case AssetTargetTypeServer:
		return "servers"
	case AssetTargetTypeTeam:
		return "teams"
	case AssetTargetTypePartner:
		return "partners"
	default:
		panic("invalid asset target type")
	}
}

func AssetTargetTypeFromTargetType(s string) (AssetTargetType, error) {
	switch s {
	case api.TargetTypeUser:
		return AssetTargetTypeUser, nil
	case api.TargetTypeBot:
		return AssetTargetTypeBot, nil
	case "server":
		return AssetTargetTypeServer, nil
	case "team":
		return AssetTargetTypeTeam, nil
	case "partner":
		return AssetTargetTypePartner, nil
	default:
		return 0, errors.New("invalid asset target type")
	}
}

// info returns the metadata of an asset given path and default path
//
// It is internal, users should be using *Info functions instead
func info(typ, path, defaultPath string) *types.AssetMetadata {
	st, err := os.Stat(state.Config.Meta.CDNPath + "/" + path)

	if err != nil {
		return &types.AssetMetadata{
			Path:        path,
			DefaultPath: defaultPath,
			Errors:      []string{"File does not exist"},
			Type:        typ,
		}
	}

	if st.IsDir() {
		return &types.AssetMetadata{
			Path:        path,
			DefaultPath: defaultPath,
			Errors:      []string{"File is a directory"},
			Type:        typ,
		}
	}

	modTime := st.ModTime()

	return &types.AssetMetadata{
		Exists:       true,
		Path:         path,
		DefaultPath:  defaultPath,
		Size:         st.Size(),
		LastModified: &modTime,
		Type:         typ,
	}
}

func BannerPath(targetType AssetTargetType, targetId string) string {
	return "banners/" + targetType.CdnString() + "/" + targetId + ".webp"
}

func BannerInfo(targetType AssetTargetType, targetId string) *types.AssetMetadata {
	return info("banner", BannerPath(targetType, targetId), "banners/default.webp")
}

// Returns the path to the avatar of the given target type and ID
func AvatarPath(targetType AssetTargetType, targetId string) string {
	return "avatars/" + targetType.CdnString() + "/" + targetId + ".webp"
}

func AvatarInfo(targetType AssetTargetType, targetId string) *types.AssetMetadata {
	return info("avatar", AvatarPath(targetType, targetId), "avatars/default.webp")
}

func ResolveAssetMetadataToUrl(t *types.AssetMetadata) string {
	if t.Exists {
		if t.LastModified == nil {
			// Use a very old time here
			t.LastModified = &time.Time{}
		}
		return state.Config.Sites.CDN + "/" + t.Path + "?ts=" + strconv.FormatInt(t.LastModified.Unix(), 10)
	} else {
		return state.Config.Sites.CDN + "/" + t.DefaultPath
	}
}

func DeleteAvatar(targetType AssetTargetType, targetId string) error {
	return DeleteFileIfExists(state.Config.Meta.CDNPath + "/avatars/" + targetType.String() + "s/" + targetId + ".webp")
}

func DeleteBanner(targetType AssetTargetType, targetId string) error {
	return DeleteFileIfExists(state.Config.Meta.CDNPath + "/banners/" + targetType.String() + "s/" + targetId + ".webp")
}
