package assets

import (
	"context"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	vanityColsArr = utils.GetCols(types.Vanity{})
	vanityCols    = strings.Join(vanityColsArr, ",")
)

func resolveImpl(ctx context.Context, code string, src string) (*types.Vanity, error) {
	var count int64

	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM vanity WHERE "+src+" = $1", code).Scan(&count)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	row, err := state.Pool.Query(ctx, "SELECT "+vanityCols+" FROM vanity WHERE "+src+" = $1", code)

	if err != nil {
		return nil, err
	}

	defer row.Close()

	var v types.Vanity

	err = pgxscan.ScanOne(&v, row)

	if err != nil {
		return nil, err
	}

	return &v, nil
}

func ResolveVanity(ctx context.Context, code string) (*types.Vanity, error) {
	var v *types.Vanity
	var err error
	for _, src := range []string{"code", "target_id"} {
		v, err = resolveImpl(ctx, code, src)

		if err != nil {
			return nil, err
		}

		if v == nil {
			continue
		}

		break
	}

	// If all fails, try checking client_id of bots
	if v == nil {
		var count int64

		err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE client_id = $1", code).Scan(&count)

		if err != nil {
			return nil, err
		}

		if count == 0 {
			return nil, nil
		}

		var botId string

		err = state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE client_id = $1", code).Scan(&botId)

		if err != nil {
			return nil, err
		}

		return resolveImpl(ctx, botId, "target_id")
	}

	return v, nil
}

func ResolveVanityByItag(ctx context.Context, itag string) (*types.Vanity, error) {
	return resolveImpl(ctx, itag, "itag")
}
