package assetmanager

import (
	"os"
	"popplio/state"
	"popplio/types"
)

type AssetTargetType int

const (
	AssetTargetTypeBots     AssetTargetType = iota
	AssetTargetTypeServers  AssetTargetType = iota
	AssetTargetTypeTeams    AssetTargetType = iota
	AssetTargetTypePartners AssetTargetType = iota
)

func (a AssetTargetType) String() string {
	switch a {
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
		return state.Config.Sites.CDN + "/" + t.Path
	} else {
		return state.Config.Sites.CDN + "/" + t.DefaultPath
	}
}
