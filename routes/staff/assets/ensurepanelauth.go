package assets

import (
	"context"
	"errors"
	"net/http"
	"popplio/state"
	"strings"
)

const (
	CapViewApps   = "ViewApps"
	CapManageApps = "ManageApps"
)

func EnsurePanelAuth(ctx context.Context, r *http.Request) (uid string, caps []string, err error) {
	ssToken := r.Header.Get("X-Staff-Auth-Token")
	loginToken := r.Header.Get("Authorization")
	userCapabilities := r.Header.Get("X-User-Capabilities")

	if ssToken == "" {
		return "", nil, errors.New("missing staff auth token normally sent by Arcadia")
	}

	if loginToken == "" {
		return "", nil, errors.New("missing authorization header")
	}

	_, err = state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE created_at < NOW() - INTERVAL '30 minutes'")

	if err != nil {
		return "", nil, err
	}

	var count int64

	err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM staffpanel__authchain WHERE token = $1", loginToken).Scan(&count)

	if err != nil {
		return "", nil, err
	}

	if count == 0 {
		return "", nil, errors.New("identityExpired")
	}

	var userId string
	var popplioToken string

	err = state.Pool.QueryRow(ctx, "SELECT user_id, popplio_token FROM staffpanel__authchain WHERE token = $1 AND state = 'active'", loginToken).Scan(&userId, &popplioToken)

	if err != nil {
		return "", nil, err
	}

	if popplioToken != ssToken {
		return "", nil, errors.New("invalid staff auth token")
	}

	return userId, strings.Split(userCapabilities, ","), nil
}
