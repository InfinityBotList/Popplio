package assets

import (
	"context"
	"errors"
	"net/http"
	"popplio/state"
)

func EnsurePanelAuth(ctx context.Context, r *http.Request) (string, error) {
	if r.Header.Get("Authorization") == "" {
		return "", errors.New("missing authorization header")
	}

	loginToken := r.Header.Get("Authorization")

	_, err := state.Pool.Exec(ctx, "DELETE FROM rpc__panelauthchain WHERE created_at < NOW() - INTERVAL '1 hour'")

	if err != nil {
		return "", err
	}

	var count int64

	err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM rpc__panelauthchain WHERE token = $1", loginToken).Scan(&count)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", errors.New("identityExpired")
	}

	var userId string

	err = state.Pool.QueryRow(ctx, "SELECT user_id FROM rpc__panelauthchain WHERE token = $1", loginToken).Scan(&userId)

	if err != nil {
		return "", err
	}

	return userId, nil
}
