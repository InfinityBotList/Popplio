// Package cdnpath holds the CDN asset name/path validators and the granular CDN
// permission check.
//
// These are the only thing standing between a staff member and the CDN root, so
// they live in a leaf package with no state dependency: that keeps them unit
// testable in isolation from the rest of the panel.
package cdnpath

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	perms "github.com/infinitybotlist/kittycat/go"
)

// NameAllowedChars is the literal allow-list from the Rust source. The duplicate
// '$' is upstream's and is harmless.
const NameAllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.:%$[](){}$@! "

// PathAllowedChars is the allow-list for asset paths.
const PathAllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.:%$/ "

// ErrBadName and ErrBadPath carry the frozen, user-visible messages.
var (
	ErrBadName = errors.New("Asset name cannot contain disallowed characters, slashes or backslashes or start with a dot")
	ErrBadPath = errors.New("Asset path cannot contain non-ASCII characters, dot-dots, doubleslashes, backslashes or start with a slash")
)

// ValidateName rejects an asset name that leaves the allowed charset, contains a
// slash or backslash, or starts with a dot.
func ValidateName(name string) error {
	for _, c := range name {
		if !strings.ContainsRune(NameAllowedChars, c) {
			return ErrBadName
		}
	}

	if strings.Contains(name, "/") || strings.Contains(name, `\`) || strings.HasPrefix(name, ".") {
		return ErrBadName
	}

	return nil
}

// ValidatePath rejects an asset path that leaves the allowed charset, contains a
// dot-dot or a double slash or a backslash, or starts with a slash.
func ValidatePath(p string) error {
	for _, c := range p {
		if !strings.ContainsRune(PathAllowedChars, c) {
			return ErrBadPath
		}
	}

	if strings.Contains(p, "..") || strings.Contains(p, "//") || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") {
		return ErrBadPath
	}

	return nil
}

// ContainedInScope reports whether target stays inside root once both are
// cleaned.
//
// HARDENING (§14b). The validators above are pure string checks; this is a
// second, structural check on the resolved path. It cannot reject a legitimate
// path: anything that stays under the scope root still passes.
func ContainedInScope(root, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)

	if cleanTarget == cleanRoot {
		return true
	}

	return strings.HasPrefix(cleanTarget, cleanRoot+string(filepath.Separator))
}

// HasPerm implements the granular CDN permission check: a caller may hold either
// the scope-specific "cdn#<scope>.<op>" or the blanket "cdn.<op>".
func HasPerm(userPerms []perms.Permission, scope, op string) bool {
	return perms.HasPerm(userPerms, perms.PermissionFromString(fmt.Sprintf("cdn#%s.%s", scope, op))) ||
		perms.HasPerm(userPerms, perms.PermissionFromString(fmt.Sprintf("cdn.%s", op)))
}
