package assets

import (
	"os"
	"popplio/state"
	"popplio/types"
)

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

	return &types.AssetMetadata{
		Exists:      true,
		Path:        path,
		DefaultPath: defaultPath,
		Size:        st.Size(),
		Type:        typ,
	}
}

func BannerInfo(targetType, targetId string) *types.AssetMetadata {
	return info("banner", "banners/"+targetType+"/"+targetId+".webp", "banners/default.webp")
}

func PartnerInfo(id string) *types.AssetMetadata {
	return info("partner", "partners/"+id+".webp", "")
}

func AvatarInfo(targetType, targetId string) *types.AssetMetadata {
	return info("partner", "avatars/"+targetType+"/"+targetId+".webp", "avatars/default.webp")
}
