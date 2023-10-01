package assets

import (
	"os"
	"popplio/state"
	"popplio/types"
)

// info returns the metadata of an asset given path and default path
//
// It is internal, users should be using *Info functions instead
func info(path, defaultPath string) *types.AssetMetadata {
	st, err := os.Stat(state.Config.Meta.CDNPath + "/" + path)

	if err != nil {
		return &types.AssetMetadata{
			DefaultPath: defaultPath,
			Errors:      []string{"File does not exist"},
		}
	}

	if st.IsDir() {
		return &types.AssetMetadata{
			DefaultPath: defaultPath,
			Errors:      []string{"File is a directory"},
		}
	}

	return &types.AssetMetadata{
		Exists:      true,
		Path:        path,
		DefaultPath: defaultPath,
		Size:        st.Size(),
	}
}

func BannerInfo(targetType, targetId string) *types.AssetMetadata {
	return info("banners/"+targetType+"/"+targetId+".webp", "images/core/banner.webp")
}
