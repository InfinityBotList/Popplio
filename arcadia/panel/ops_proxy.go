package panel

import (
	"context"
	"io"
	"net/http"
	"strings"

	"popplio/arcadia/types"
	"popplio/state"
)

// The signed reverse proxy into Popplio's own staff API.
//
// The panel cannot call Popplio directly — it has no credential of its own — so
// requests are relayed from here with the instance's signature, and the upstream
// status and body come back verbatim.

// popplioStaff is a signed reverse proxy into Popplio's staff API.
func (s *Server) popplioStaff(ctx context.Context, q *types.QPopplioStaff) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !validHTTPMethod(q.Method) {
		return writeText(http.StatusBadRequest, "Invalid method"), nil
	}

	if !strings.HasPrefix(q.Path, "/") {
		return writeText(http.StatusBadRequest, "Path must start with /"), nil
	}

	// HARDENING (§14b): upstream only checks the leading '/', so a path of
	// "//evil.example" or one containing ".." could retarget the request at a
	// different host or escape the API base. We resolve the path against the base
	// URL and reject anything that leaves it. Legitimate paths (a plain absolute
	// path, optionally with a query string) are unaffected.
	target, err := safeJoinPopplio(q.Path)

	if err != nil {
		return writeText(http.StatusBadRequest, "Path must start with /"), nil
	}

	var popplioToken string

	err = state.Pool.QueryRow(ctx, "SELECT popplio_token FROM staffpanel__authchain WHERE token = $1", q.LoginToken).Scan(&popplioToken)

	if err != nil {
		return response{}, newError(err)
	}

	req, err := http.NewRequestWithContext(ctx, q.Method, target, strings.NewReader(q.Body))

	if err != nil {
		return response{}, newError(err)
	}

	req.Header.Set("User-Agent", "arcadia")
	// QUIRK (reproduced): the request PATH is sent as X-Forwarded-For. This looks
	// like a copy/paste bug but downstream may rely on it. See CONFORMANCE.md.
	req.Header.Set("X-Forwarded-For", q.Path)
	req.Header.Set("X-Staff-Auth-Token", popplioToken)
	req.Header.Set("X-User-ID", authData.UserID)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return response{}, newError(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return response{}, newError(err)
	}

	// Relay the upstream status and body verbatim.
	return writeText(resp.StatusCode, string(body)), nil
}

func validHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect,
		http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
