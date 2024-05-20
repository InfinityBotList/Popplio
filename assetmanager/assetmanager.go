package assetmanager

import (
	"errors"
	"os"
	"popplio/state"
	"popplio/types"
	"strconv"
	"time"
)

type AssetTargetType int

const (
	AssetTargetTypeUsers    AssetTargetType = iota
	AssetTargetTypeBots     AssetTargetType = iota
	AssetTargetTypeServers  AssetTargetType = iota
	AssetTargetTypeTeams    AssetTargetType = iota
	AssetTargetTypePartners AssetTargetType = iota
)

func (a AssetTargetType) String() string {
	switch a {
	case AssetTargetTypeUsers:
		return "users"
	case AssetTargetTypeBots:
		return "bots"
	case AssetTargetTypeServers:
		return "servers"
	case AssetTargetTypeTeams:
		return "teams"
	case AssetTargetTypePartners:
		return "partners"
	default:
		panic("invalid asset target type")
	}
}

func AssetTargetTypeFromString(s string) (AssetTargetType, error) {
	switch s {
	case "users":
		return AssetTargetTypeUsers, nil
	case "bots":
		return AssetTargetTypeBots, nil
	case "servers":
		return AssetTargetTypeServers, nil
	case "teams":
		return AssetTargetTypeTeams, nil
	case "partners":
		return AssetTargetTypePartners, nil
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
			DefaultPath: defaultPath,
			Errors:      []string{"File does not exist"},
			Type:        typ,
		}
	}

	if st.IsDir() {
		return &types.AssetMetadata{
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

func BannerInfo(targetType AssetTargetType, targetId string) *types.AssetMetadata {
	return info("banner", "banners/"+targetType.String()+"/"+targetId+".webp", "banners/default.webp")
}

func AvatarInfo(targetType AssetTargetType, targetId string) *types.AssetMetadata {
	return info("avatar", "avatars/"+targetType.String()+"/"+targetId+".webp", "avatars/default.webp")
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
