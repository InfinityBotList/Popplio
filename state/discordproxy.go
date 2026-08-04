package state

import (
	"net/http"

	"github.com/disgoorg/disgo/rest"
)

// proxyAuthTransport adds X-Upstream-Authorization to every outgoing REST
// request, carrying the specific bot's own token.
//
// Config.Meta.PopplioProxy (gateway.nodebyte.host) strips whatever
// Authorization header a caller sends and replaces it with its own shared
// bot credential, so callers that need to authenticate as their own
// distinct bot application pass their token via X-Upstream-Authorization
// instead — the gateway's `discord` service is opted in
// (allowCallerOverride) to honor that header and forward it as the real
// Authorization header sent to Discord.
type proxyAuthTransport struct {
	token string
	base  http.RoundTripper
}

func (t *proxyAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Upstream-Authorization", "Bot "+t.token)
	return t.base.RoundTrip(req)
}

// ProxyRestOpts returns rest.ConfigOpt values that route a bot.Client's REST
// traffic through Config.Meta.PopplioProxy, authenticating as the given
// bot's own token rather than the gateway's shared credential.
//
// Used by every distinct bot identity in this codebase (Popplio's own bot in
// Setup, Arcadia's staff bot in arcadia/dclient) so each keeps its own
// Discord application identity while still sharing the gateway's
// rate-limiting and routing.
func ProxyRestOpts(token string) []rest.ConfigOpt {
	return []rest.ConfigOpt{
		rest.WithURL(Config.Meta.PopplioProxy + "/v10"),
		rest.WithHTTPClient(&http.Client{
			Transport: &proxyAuthTransport{token: token, base: http.DefaultTransport},
		}),
	}
}
