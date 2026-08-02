package panel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// authVersion is the protocol version the panel must send on Authorize.
const authVersion = 5

// oauthUserAgent is sent on both Discord calls.
const oauthUserAgent = "DiscordBot (arcadia v1.0)"

// authorize handles the login flow. It takes no prior authentication.
func (s *Server) authorize(ctx context.Context, q *types.QAuthorize) (response, error) {
	if q.Version != authVersion {
		return writeText(http.StatusBadRequest, "Invalid version"), nil
	}

	switch {
	case q.Action.Begin != nil:
		return s.authBegin(q.Action.Begin)
	case q.Action.CreateSession != nil:
		return s.authCreateSession(ctx, q.Action.CreateSession)
	case q.Action.CheckMfaState != nil:
		return s.authCheckMfaState(ctx, q.Action.CheckMfaState)
	case q.Action.ResetMfaTotp != nil:
		return s.authResetMfaTotp(ctx, q.Action.ResetMfaTotp)
	case q.Action.ActivateSession != nil:
		return s.authActivateSession(ctx, q.Action.ActivateSession)
	case q.Action.Logout != nil:
		return s.authLogout(ctx, q.Action.Logout)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No authorize action was specified")
	}
}

func (s *Server) authBegin(action *types.AuthBegin) (response, error) {
	if action.Scope != state.Config.Arcadia.Panel.PanelScope {
		return writeText(http.StatusBadRequest, "Invalid scope"), nil
	}

	// NOTE: redirect_url is neither validated nor URL-encoded here. Preserved
	// byte-for-byte; see CONFORMANCE.md.
	loginURL := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify",
		state.Config.Arcadia.Panel.ClientID, action.RedirectURL,
	)

	return writeJSON(http.StatusOK, types.StartAuth{
		LoginURL:      loginURL,
		Scope:         state.Config.Arcadia.Panel.PanelScope,
		ResponseScope: state.Config.Arcadia.Panel.PanelResponseScope,
	}), nil
}

func (s *Server) authCreateSession(ctx context.Context, action *types.AuthCreateSession) (response, error) {
	if !containsString(state.Config.Arcadia.Panel.RedirectURL, action.RedirectURL) {
		return writeText(http.StatusBadRequest, "Invalid redirect url"), nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	form := url.Values{}
	form.Set("client_id", state.Config.Arcadia.Panel.ClientID)
	form.Set("client_secret", state.Config.Arcadia.Panel.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", action.Code)
	form.Set("redirect_uri", action.RedirectURL)
	form.Set("scope", "identify")

	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))

	if err != nil {
		return response{}, newError(err)
	}

	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("User-Agent", oauthUserAgent)

	tokenResp, err := client.Do(tokenReq)

	if err != nil {
		return response{}, newError(err)
	}

	defer tokenResp.Body.Close()

	if tokenResp.StatusCode < 200 || tokenResp.StatusCode > 299 {
		return response{}, newError(fmt.Errorf("HTTP status client error (%d %s) for url (https://discord.com/api/oauth2/token)", tokenResp.StatusCode, http.StatusText(tokenResp.StatusCode)))
	}

	var oauth2 struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(tokenResp.Body).Decode(&oauth2); err != nil {
		return response{}, newError(err)
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)

	if err != nil {
		return response{}, newError(err)
	}

	userReq.Header.Set("Authorization", "Bearer "+oauth2.AccessToken)
	userReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	userReq.Header.Set("User-Agent", oauthUserAgent)

	userResp, err := client.Do(userReq)

	if err != nil {
		return response{}, newError(err)
	}

	defer userResp.Body.Close()

	if userResp.StatusCode < 200 || userResp.StatusCode > 299 {
		return response{}, newError(fmt.Errorf("HTTP status client error (%d %s) for url (https://discord.com/api/users/@me)", userResp.StatusCode, http.StatusText(userResp.StatusCode)))
	}

	var user struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		return response{}, newError(err)
	}

	var positions []string

	err = state.Pool.QueryRow(ctx, "SELECT positions FROM staff_members WHERE user_id = $1", user.ID).Scan(&positions)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return writeText(http.StatusForbidden, "You are not a staff member [not in db]"), nil
		}

		return response{}, newError(err)
	}

	if len(positions) == 0 {
		return writeText(http.StatusForbidden, "You are not a staff member [no positions]"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE user_id = $1", user.ID); err != nil {
		return response{}, newError(err)
	}

	// Token length is a random value in [4196, 6000).
	token := impls.GenRandom(impls.RandRange(4196, 6000))

	_, err = tx.Exec(ctx,
		"INSERT INTO staffpanel__authchain (user_id, token, popplio_token, state) VALUES ($1, $2, $3, $4)",
		user.ID, token, impls.GenRandom(2048), "pending")

	if err != nil {
		return response{}, newError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	// The raw token is the whole body.
	return writeText(http.StatusOK, token), nil
}

func (s *Server) authCheckMfaState(ctx context.Context, action *types.AuthCheckMfaState) (response, error) {
	// This is the only endpoint that works on both pending and active sessions.
	authData, err := checkAuthInsecure(ctx, action.LoginToken)

	if err != nil {
		return response{}, err
	}

	if authData.State != "pending" && authData.State != "active" {
		return response{}, errStatus(http.StatusBadRequest, "This endpoint can only be used by pending and active sessions")
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var (
		mfaSecret   *string
		mfaVerified bool
	)

	err = tx.QueryRow(ctx, "SELECT mfa_secret, mfa_verified FROM staff_members WHERE user_id = $1", authData.UserID).Scan(&mfaSecret, &mfaVerified)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return response{}, errStatus(http.StatusBadRequest, "You are not a staff member")
		}

		return response{}, newError(fmt.Errorf("Failed to fetch staff member mfa_secret/mfa_verified: %s", err))
	}

	if mfaSecret != nil && mfaVerified {
		// Already enrolled: nothing to hand back.
		return writeJSON(http.StatusOK, types.MfaLogin{Info: nil}), nil
	}

	secret, err := generateTOTPSecret()

	if err != nil {
		return response{}, newError(err)
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_members SET mfa_secret = $1 WHERE user_id = $2", secret, authData.UserID); err != nil {
		return response{}, newError(err)
	}

	otpURL := otpAuthURI(secret)

	qr, err := renderQRSVG(otpURL)

	if err != nil {
		return response{}, newError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, types.MfaLogin{Info: &types.MfaLoginSecret{
		Secret: secret,
		OtpURL: otpURL,
		QRCode: qr,
	}}), nil
}

func (s *Server) authResetMfaTotp(ctx context.Context, action *types.AuthResetMfaTotp) (response, error) {
	authData, err := checkAuth(ctx, action.LoginToken)

	if err != nil {
		return response{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var mfaSecret *string

	if err := tx.QueryRow(ctx, "SELECT mfa_secret FROM staff_members WHERE user_id = $1", authData.UserID).Scan(&mfaSecret); err != nil {
		return response{}, newError(err)
	}

	if mfaSecret == nil {
		return response{}, errStatus(http.StatusBadRequest, "mfaNotSetup")
	}

	valid, err := verifyTOTP(action.OTP, *mfaSecret)

	if err != nil || !valid {
		// Capital "Entered" here; ActivateSession uses lowercase. Both reproduced.
		return response{}, errStatus(http.StatusBadRequest, "Invalid OTP Entered")
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_members SET mfa_secret = NULL, mfa_verified = FALSE WHERE user_id = $1", authData.UserID); err != nil {
		return response{}, newError(err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE user_id = $1", authData.UserID); err != nil {
		return response{}, newError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) authActivateSession(ctx context.Context, action *types.AuthActivateSession) (response, error) {
	authData, err := checkAuthInsecure(ctx, action.LoginToken)

	if err != nil {
		return response{}, err
	}

	if authData.State != "pending" {
		return response{}, errStatus(http.StatusBadRequest, "sessionAlreadyActive")
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var (
		mfaSecret   *string
		mfaVerified bool
	)

	err = tx.QueryRow(ctx, "SELECT mfa_secret, mfa_verified FROM staff_members WHERE user_id = $1", authData.UserID).Scan(&mfaSecret, &mfaVerified)

	if err != nil {
		return response{}, newError(err)
	}

	if mfaSecret == nil {
		return response{}, errStatus(http.StatusBadRequest, "mfaNotSetup")
	}

	valid, err := verifyTOTP(action.OTP, *mfaSecret)

	if err != nil || !valid {
		// Lowercase "entered" here; ResetMfaTotp uses uppercase. Both reproduced.
		return response{}, errStatus(http.StatusBadRequest, "Invalid OTP entered")
	}

	if _, err := tx.Exec(ctx, "UPDATE staffpanel__authchain SET state = 'active' WHERE token = $1", action.LoginToken); err != nil {
		return response{}, newError(err)
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_members SET mfa_verified = TRUE WHERE user_id = $1", authData.UserID); err != nil {
		return response{}, newError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) authLogout(ctx context.Context, action *types.AuthLogout) (response, error) {
	// No auth check at all: deleting a token that does not exist is not an error.
	tag, err := state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE token = $1", action.LoginToken)

	if err != nil {
		return response{}, newError(err)
	}

	// The affected row count is the whole body.
	return writeText(http.StatusOK, strconv.FormatInt(tag.RowsAffected(), 10)), nil
}

func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}

	return false
}
