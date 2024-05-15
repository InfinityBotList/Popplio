package hooks

import (
	"errors"
	"fmt"
	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"
	"popplio/webhooks/sender"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexServerColsArr = db.GetCols(types.IndexServer{})
	indexServerCols    = strings.Join(indexServerColsArr, ", ")
)

type ServerDriver struct{}

func (sd ServerDriver) TargetType() string {
	return "server"
}

func (sd ServerDriver) Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error) {
	row, err := state.Pool.Query(state.Context, "SELECT "+indexServerCols+" FROM servers WHERE server_id = $1", id)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errors.New("server not found")
	}

	if err != nil {
		state.Logger.Error("Failed to fetch server for this hook", zap.Error(err), zap.String("serverID", id), zap.String("userID", userId))
		return nil, nil, err
	}

	server, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		state.Logger.Error("Failed to fetch server data for this hook", zap.Error(err), zap.String("serverID", id), zap.String("userID", userId))
		return nil, nil, err
	}

	server.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeServers, server.ServerID)
	server.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeServers, server.ServerID)

	var code string

	err = state.Pool.QueryRow(state.Context, "SELECT code FROM vanity WHERE itag = $1", server.VanityRef).Scan(&code)

	if err != nil {
		return nil, nil, fmt.Errorf("error while getting server vanity code [db fetch]: %w", err)
	}

	server.Vanity = code

	targets := events.Target{
		Server: &server,
	}

	entity := sender.WebhookEntity{
		EntityID:   server.ServerID,
		EntityName: server.Name,
		EntityType: sd.TargetType(),
	}

	return &targets, &entity, nil
}

func (sd ServerDriver) CanBeConstructed(userId, targetId string) (bool, error) {
	return true, nil
}

func (sd ServerDriver) SupportsPullPending(userId, targetId string) (bool, error) {
	return true, nil
}

func init() {
	drivers.RegisterDriver(ServerDriver{})
}
