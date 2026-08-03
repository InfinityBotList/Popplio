// Package create_oauth2_login implements POST /auth/login/discord-oauth2 —
// "Create OAuth2 Login".
//
// Takes in a `code` query parameter and returns a user `token`.
package create_oauth2_login

import (
	"fmt"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/ratelimit"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
	ua "github.com/mileusna/useragent"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.AuthorizeRequest{})
)

// supportedProtocol is the client protocol this endpoint speaks. Clients send
// their protocol so an outdated one gets a clear message rather than a confusing
// failure further down the exchange.
const supportedProtocol = "persepolis-infernoplex"

// codeReuseWindow is how long a used authorization code is remembered.
//
// Discord rejects a replayed code itself; remembering codes here means a replay
// is refused before it costs a round trip, and closes the window in which two
// concurrent requests could both exchange the same code.
const codeReuseWindow = 5 * time.Minute

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create OAuth2 Login",
		Description: "Takes in a ``code`` query parameter and returns a user ``token``. **Cannot be used outside of the site for security reasons but documented in case we wish to allow its use in the future.**",
		Req:         types.AuthorizeRequest{},
		Resp:        types.CreateSessionResponse{},
	}
}

// validateRequest checks the parts of the request that can be rejected without
// talking to Discord.
//
// These are all misconfigured- or outdated-client conditions rather than user
// error, which is why each gets its own message. Failures are reported as
// [oauthError] so the handler can route them through oauthFailure alongside
// every other failure in the exchange.
func validateRequest(req types.AuthorizeRequest) error {
	switch {
	case req.Protocol != supportedProtocol:
		return &oauthError{
			status:  http.StatusBadRequest,
			message: "Your client is outdated and is not supported. Please contact the developers of this client.",
			reason:  "Login attempt with unsupported client protocol",
		}
	case !slices.Contains(state.Config.DiscordAuth.AllowedRedirects, req.RedirectURI):
		return &oauthError{
			status:  http.StatusBadRequest,
			message: "Malformed redirect_uri",
			reason:  "Login attempt with disallowed redirect_uri",
		}
	case req.ClientID != state.Config.DiscordAuth.ClientID:
		return &oauthError{
			status:  http.StatusBadRequest,
			message: "Misconfigured client! Client id is incorrect",
			reason:  "Login attempt with incorrect client id",
		}
	}

	return nil
}

// sessionName describes the client a session was created from, so users can
// recognise their own devices in their session list.
//
// A missing User-Agent falls back to a generic desktop Chrome string rather than
// an empty name, so the session list never shows a blank entry.
func sessionName(userAgent string) string {
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
	}

	parsed := ua.Parse(userAgent)

	return fmt.Sprintf("%s (on %s %s) [mobile: %t]", parsed.Name, parsed.OS, parsed.Version, parsed.Mobile)
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "login",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "login"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	// Every response below carries the ratelimit headers, so clients can see
	// their remaining budget on failures as well as successes.
	headers := limit.Headers()

	var req types.AuthorizeRequest

	hresp, ok := uapi.MarshalReqWithHeaders(r, &req, headers)

	if !ok {
		return hresp
	}

	err = state.Validator.Struct(req)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if err := validateRequest(req); err != nil {
		return oauthFailure(err, headers)
	}

	// SetNX marks the code used and checks whether it was already used in a
	// single atomic round trip — a separate Exists+Set here would leave a
	// window where two concurrent requests carrying the same code could both
	// pass the check before either marked it used.
	isNew, err := state.Redis.SetNX(d.Context, "codecache:"+req.Code, "0", codeReuseWindow).Result()

	if err != nil {
		return resp.Err("Error while checking code reuse", err)
	}

	if !isNew {
		return resp.WithHeaders(resp.BadRequest("Code has been clearly used before and is as such invalid"), headers)
	}

	accessToken, err := exchangeCodeForToken(d.Context, req.Code, req.RedirectURI)

	if err != nil {
		return oauthFailure(err, headers)
	}

	user, err := fetchOauthUser(d.Context, accessToken)

	if err != nil {
		return oauthFailure(err, headers)
	}

	existed, err := ensureUser(d.Context, user.ID)

	if err != nil {
		return oauthFailure(err, headers)
	}

	go sendAuthLog(user, req, !existed)

	// Only an existing user can carry a ban, so a first-time login skips the
	// check entirely.
	if existed {
		if err := checkBanScope(d.Context, user.ID, req.Scope); err != nil {
			return oauthFailure(err, headers)
		}
	}

	token, sessionID, err := createSession(d.Context, user.ID, sessionName(r.UserAgent()))

	if err != nil {
		return oauthFailure(err, headers)
	}

	return resp.WithHeaders(resp.OK(types.CreateSessionResponse{
		TargetID:  user.ID,
		Token:     token,
		SessionID: sessionID,
	}), headers)
}
