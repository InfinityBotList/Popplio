package assets

import (
	"context"
	"errors"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	vanityColsArr = db.GetCols(types.Vanity{})
	vanityCols    = strings.Join(vanityColsArr, ",")
)

func resolveImpl(ctx context.Context, code string, src string) (*types.Vanity, error) {
	row, err := state.Pool.Query(ctx, "SELECT "+vanityCols+" FROM vanity WHERE "+src+" = $1", code)

	if err != nil {
		return nil, err
	}

	v, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Vanity])

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &v, nil
}

func ResolveVanity(ctx context.Context, code string) (*types.Vanity, error) {
	// First check bot_id and client_id to avoid vanity stealing
	var botId string

	err := state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE client_id = $1", code).Scan(&botId)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if botId != "" {
		return resolveImpl(ctx, botId, "target_id")
	}

	// Then check server id
	var serverId string

	err = state.Pool.QueryRow(ctx, "SELECT server_id FROM servers WHERE server_id = $1", code).Scan(&serverId)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if serverId != "" {
		return resolveImpl(ctx, serverId, "target_id")
	}

	var v *types.Vanity
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

	return v, nil
}

func ResolveVanityByItag(ctx context.Context, itag string) (*types.Vanity, error) {
	return resolveImpl(ctx, itag, "itag")
}
