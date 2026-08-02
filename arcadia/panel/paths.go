package panel

import (
	"errors"
	"net/url"
	"path"
	"strings"

	"popplio/arcadia/cdnpath"
	"popplio/state"
)

// safeJoinPopplio resolves a caller-supplied path against Popplio's base URL and
// refuses anything that would leave it.
//
// HARDENING (§14b). Upstream builds the target as a bare string concatenation
// after checking only that the path starts with '/', so "//host/x" (a
// protocol-relative URL) retargets the request at an arbitrary host and ".."
// segments escape the API base. Legitimate input - an absolute path with an
// optional query string - resolves identically here.
func safeJoinPopplio(rawPath string) (string, error) {
	base, err := url.Parse(state.Config.Sites.API.Parse())

	if err != nil {
		return "", err
	}

	ref, err := url.Parse(rawPath)

	if err != nil {
		return "", err
	}

	// A parsed reference that carries a scheme or a host is not a path.
	if ref.Scheme != "" || ref.Host != "" || ref.Opaque != "" {
		return "", errors.New("path must not contain a scheme or host")
	}

	if !strings.HasPrefix(ref.Path, "/") {
		return "", errors.New("path must start with /")
	}

	resolved := base.ResolveReference(ref)

	if resolved.Scheme != base.Scheme || resolved.Host != base.Host {
		return "", errors.New("path escapes the popplio base url")
	}

	basePath := base.EscapedPath()
	if basePath == "" {
		basePath = "/"
	}

	if !strings.HasPrefix(path.Clean(resolved.EscapedPath())+"/", path.Clean(basePath)+"/") {
		return "", errors.New("path escapes the popplio base url")
	}

	return resolved.String(), nil
}

// containedInScope delegates to the leaf package so the check is unit tested
// there.
func containedInScope(root, target string) bool {
	return cdnpath.ContainedInScope(root, target)
}
