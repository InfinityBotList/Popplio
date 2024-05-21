package assets

import (
	"context"
	"fmt"
	"popplio/assetmanager"
	"popplio/state"
	"popplio/types"
)

func ResolveIndexServer(ctx context.Context, server *types.IndexServer) error {
	var code string

	err := state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", server.VanityRef).Scan(&code)

	if err != nil {
		return fmt.Errorf("error querying vanity table: %w", err)
	}

	server.Vanity = code
	server.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeServers, server.ServerID)
	server.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeServers, server.ServerID)
	return nil
}
