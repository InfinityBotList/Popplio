package hooks

import (
	"errors"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"
	"popplio/webhooks/sender"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type ServerDriver struct{}

func (sd ServerDriver) TargetType() string {
	return "server"
}

func (sd ServerDriver) Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error) {
	var name, avatar, short string

	err := state.Pool.QueryRow(state.Context, "SELECT name, avatar, short FROM servers WHERE server_id = $1", id).Scan(&name, &avatar, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errors.New("server not found")
	}

	if err != nil {
		state.Logger.Error("Failed to fetch name/avatar/short of server for this serverhook", zap.Error(err), zap.String("serverID", id), zap.String("userID", userId))
		return nil, nil, err
	}

	targets := events.Target{
		Server: &types.SEO{
			Name:   name,
			ID:     id,
			Avatar: avatar,
			Short:  short,
		},
	}
	entity := sender.WebhookEntity{
		EntityID:   id,
		EntityName: name,
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
	drivers.RegisterCoreWebhook(ServerDriver{})
}
