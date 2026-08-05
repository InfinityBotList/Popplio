package sorbet

import (
	"context"
	"time"

	"popplio/state"

	"github.com/jackc/pgx/v5"
)

type session struct {
	ID         string
	Name       *string
	CreatedAt  time.Time
	Type       string
	TargetType string
	TargetID   string
	PermLimits []string
	Expiry     time.Time
}

func clearExpiredSessions(ctx context.Context) error {
	_, err := state.Pool.Exec(ctx, "DELETE FROM api_sessions WHERE expiry < NOW()")
	return err
}

func sessionFromToken(ctx context.Context, token string) (*session, error) {
	if err := clearExpiredSessions(ctx); err != nil {
		return nil, err
	}

	var s session

	err := state.Pool.QueryRow(ctx,
		"SELECT id, name, created_at, type, target_type, target_id, perm_limits, expiry FROM api_sessions WHERE token = $1",
		token,
	).Scan(&s.ID, &s.Name, &s.CreatedAt, &s.Type, &s.TargetType, &s.TargetID, &s.PermLimits, &s.Expiry)

	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &s, nil
}
