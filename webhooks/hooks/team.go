package hooks

import (
	"errors"
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
	teamColsArr = db.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ", ")
)

type TeamDriver struct{}

func (td TeamDriver) TargetType() string {
	return "team"
}

func (td TeamDriver) Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error) {
	row, err := state.Pool.Query(state.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", id)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errors.New("team not found")
	}

	if err != nil {
		state.Logger.Error("Failed to fetch team data for this teambook", zap.Error(err), zap.String("teamID", id), zap.String("userID", userId))
		return nil, nil, err
	}

	team, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

	if err != nil {
		state.Logger.Error("Failed to fetch team data for this teambook", zap.Error(err), zap.String("teamID", id), zap.String("userID", userId))
		return nil, nil, err
	}

	team.Entities = &types.TeamEntities{
		Targets: []string{}, // We don't provide any entities right now, may change
	}

	team.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeTeams, team.ID)
	team.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, team.ID)

	targets := events.Target{
		Team: &team,
	}
	entity := sender.WebhookEntity{
		EntityID:   team.ID,
		EntityName: team.Name,
		EntityType: td.TargetType(),
	}

	return &targets, &entity, nil
}

func (td TeamDriver) CanBeConstructed(userId, targetId string) (bool, error) {
	return true, nil
}

func (td TeamDriver) SupportsPullPending(userId, targetId string) (bool, error) {
	return true, nil
}

func init() {
	drivers.RegisterCoreWebhook(TeamDriver{})
}
