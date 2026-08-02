package create_oauth2_login

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
)

// oauthUser is the subset of Discord's /users/@me payload this endpoint needs.
type oauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
}

// oauthError is a failure somewhere in the Discord OAuth2 exchange, carrying
// everything needed to turn it into an HTTP response.
//
// The exchange fails in a dozen places, and each one wants a different status
// and a different sentence for the caller — but the mapping is a property of the
// failure, not of the handler. Keeping status and message on the error lets the
// exchange helpers stay free of HTTP concerns while the handler stays free of
// per-stage error mapping.
type oauthError struct {
	// status is the HTTP status this failure maps to. Failures caused by the
	// caller's input (a replayed or invalid code) are 4xx; failures of the
	// Discord call itself are 5xx.
	status int
	// message is the client-facing text. It never contains the underlying
	// error, which is for the log only.
	message string
	// reason is the log line, naming the stage that failed.
	reason string
	// cause is the underlying error, if the failure came from one.
	cause error
}

func (e *oauthError) Error() string {
	if e.cause != nil {
		return e.reason + ": " + e.cause.Error()
	}

	return e.reason
}

func (e *oauthError) Unwrap() error { return e.cause }

// response renders the error, logging 5xx faults and staying silent on 4xx, in
// line with the conventions in [resp].
func (e *oauthError) response() uapi.HttpResponse {
	if e.status >= http.StatusInternalServerError {
		return resp.ErrBody(e.reason, e.message, e.cause)
	}

	return resp.Status(e.status, e.message)
}

// oauthFailure turns any error from the exchange helpers into a response,
// attaching headers so the rate limit is reported on failures too.
//
// A non-oauthError is treated as an unexpected internal fault rather than
// silently rendered, so a future helper that forgets to wrap still fails safe.
func oauthFailure(err error, headers map[string]string) uapi.HttpResponse {
	oerr, ok := err.(*oauthError)
	if !ok {
		return resp.WithHeaders(resp.Err("Unexpected error during oauth2 exchange", err), headers)
	}

	return resp.WithHeaders(oerr.response(), headers)
}

// discordAPI is the Discord API base this endpoint talks to.
const discordAPI = "https://discord.com/api/v10"

// exchangeCodeForToken trades an OAuth2 authorization code for a user access
// token.
//
// The code is single-use: Discord rejects a replay, and the caller is expected
// to have already rejected codes seen recently (see the codecache check in
// Route) so a replay never reaches Discord at all.
func exchangeCodeForToken(ctx context.Context, code, redirectURI string) (string, error) {
	httpResp, err := http.PostForm(discordAPI+"/oauth2/token", url.Values{
		"client_id":     {state.Config.DiscordAuth.ClientID},
		"client_secret": {state.Config.DiscordAuth.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"scope":         {"identify"},
	})

	if err != nil {
		return "", &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to send token request to Discord",
			reason:  "Failed to send oauth2 token request to discord",
			cause:   err,
		}
	}

	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)

	if err != nil {
		return "", &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to read token response from Discord",
			reason:  "Failed to read oauth2 token response from discord",
			cause:   err,
		}
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}

	err = jsonimpl.Unmarshal(body, &token)

	if err != nil {
		// A body Discord won't let us parse most often means it rejected the
		// code, so this is the caller's problem rather than ours.
		return "", &oauthError{
			status:  http.StatusBadRequest,
			message: "Failed to parse token response from Discord",
			reason:  "Failed to parse oauth2 token response from discord",
			cause:   err,
		}
	}

	if token.AccessToken == "" {
		return "", &oauthError{
			status:  http.StatusBadRequest,
			message: "No access token provided by Discord",
			reason:  "No access token provided by discord",
		}
	}

	return token.AccessToken, nil
}

// fetchOauthUser resolves the Discord user that an access token belongs to.
func fetchOauthUser(ctx context.Context, accessToken string) (oauthUser, error) {
	var user oauthUser

	httpReq, err := http.NewRequestWithContext(ctx, "GET", discordAPI+"/users/@me", nil)

	if err != nil {
		return user, &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to create request to Discord to fetch user info",
			reason:  "Failed to create oauth2 request to discord",
			cause:   err,
		}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)

	httpResp, err := http.DefaultClient.Do(httpReq)

	if err != nil {
		return user, &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to send oauth2 request to Discord",
			reason:  "Failed to send oauth2 request to discord",
			cause:   err,
		}
	}

	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)

	if err != nil {
		return user, &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to read oauth2 response from Discord",
			reason:  "Failed to read oauth2 response from discord",
			cause:   err,
		}
	}

	err = jsonimpl.Unmarshal(body, &user)

	if err != nil {
		return user, &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to parse oauth2 response from Discord",
			reason:  "Failed to parse oauth2 response from discord",
			cause:   err,
		}
	}

	if user.ID == "" {
		// Discord answered, but with no user — the token was not valid for
		// identify, which traces back to the code the caller sent.
		return user, &oauthError{
			status:  http.StatusBadRequest,
			message: "No user ID provided by Discord. Invalid code/access token?",
			reason:  "No user ID provided by discord. Invalid code/access token?",
		}
	}

	return user, nil
}

// ensureUser creates the users row for a first-time login, reporting whether the
// user already existed.
//
// The caller needs the distinction for more than logging: a brand-new user
// cannot be banned, so the ban check is skipped for them.
func ensureUser(ctx context.Context, userID string) (existed bool, err error) {
	err = state.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", userID).Scan(&existed)

	if err != nil {
		return false, &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to check if user exists on database",
			reason:  "Failed to check if user exists on database",
			cause:   err,
		}
	}

	if existed {
		return true, nil
	}

	_, err = state.Pool.Exec(
		ctx,
		"INSERT INTO users (user_id, extra_links, developer, certified) VALUES ($1, $2, false, false)",
		userID,
		[]types.Link{},
	)

	if err != nil {
		return false, &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to create user on database",
			reason:  "Failed to create user on database",
			cause:   err,
		}
	}

	return false, nil
}

// checkBanScope enforces the relationship between a user's ban state and the
// scope they asked for.
//
// The two are deliberately coupled: a banned user may only log in through the
// ban_exempt scope (which the frontend uses to show them the appeal flow), and
// an unbanned user may not use that scope at all, since it would otherwise be a
// way to obtain a session with different semantics than intended.
func checkBanScope(ctx context.Context, userID, scope string) error {
	var banned bool

	err := state.Pool.QueryRow(ctx, "SELECT banned FROM users WHERE user_id = $1", userID).Scan(&banned)

	if err != nil {
		return &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to get API token from database",
			reason:  "Failed to get API token from database",
			cause:   err,
		}
	}

	if banned && scope != "ban_exempt" {
		return &oauthError{
			status:  http.StatusForbidden,
			message: "You are banned from the list. If you think this is a mistake, please contact support.",
			reason:  "Banned user attempted login outside ban_exempt scope",
		}
	}

	if !banned && scope == "ban_exempt" {
		return &oauthError{
			status:  http.StatusForbidden,
			message: "The selected scope is not allowed for unbanned users [ban_exempt].",
			reason:  "Unbanned user attempted login with ban_exempt scope",
		}
	}

	return nil
}

// createSession issues a login session for a user and returns its token and id.
//
// sessionName is a human-readable description of the client, shown to the user
// in their session list so they can tell their devices apart.
//
// Session lifetime here must stay in sync with the frontend's client-side
// expires_at estimate (see auth.ts's callback()) since this endpoint does not
// return an expiry timestamp for the frontend to read.
func createSession(ctx context.Context, userID, sessionName string) (token, sessionID string, err error) {
	token = crypto.RandString(128)

	err = state.Pool.QueryRow(
		ctx,
		"INSERT INTO api_sessions (target_type, target_id, type, token, expiry, name) VALUES ('user', $1, 'login', $2, NOW() + INTERVAL '30 days', $3) RETURNING id",
		userID, token, sessionName,
	).Scan(&sessionID)

	if err != nil {
		return "", "", &oauthError{
			status:  http.StatusInternalServerError,
			message: "Failed to create session token",
			reason:  "Failed to create session token",
			cause:   err,
		}
	}

	return token, sessionID, nil
}
