package assets

import (
	"context"
	"errors"
	"net/http"
	"popplio/state"
)

func EnsurePanelAuth(ctx context.Context, r *http.Request) (uid string, err error) {
	ssToken := r.Header.Get("X-Staff-Auth-Token")
	userId := r.Header.Get("X-User-ID")

	if ssToken == "" {
		return "", errors.New("missing staff auth token normally sent by Arcadia")
	}

	if ssToken == "" {
		return "", errors.New("missing authorization header")
	}

	if userId == "" {
		return "", errors.New("missing user id header")
	}

	var count int64

	err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM staffpanel__authchain WHERE popplio_token = $1 AND user_id = $2", ssToken, userId).Scan(&count)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", errors.New("identityExpired")
	}

	return userId, nil
}
