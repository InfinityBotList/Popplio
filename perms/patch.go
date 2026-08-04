package perms

import (
	"errors"
	"fmt"
)

// ErrCannotManage is returned when someone tries to grant or revoke a
// permission they do not hold themselves. Callers match on it to turn the
// failure into a 403.
var ErrCannotManage = errors.New("insufficient permission to manage this permission")

// CheckPatch reports whether a manager may change a permission set from current
// into next.
//
// The rule is that you cannot give away, or take away, power you do not have:
// every permission that differs between the two must be one the manager holds.
// Holding the domain's super permission satisfies this for everything.
//
// Rank is checked separately by the caller, since it is a fact about the role
// or member being edited rather than about the permissions themselves.
func CheckPatch(manager, current, next Set) error {
	for _, p := range current.Diff(next) {
		if !manager.Has(p) {
			return fmt.Errorf("%w: %s", ErrCannotManage, p)
		}
	}

	return nil
}

// CanGrant reports whether a manager may hand out a single permission.
func CanGrant(manager Set, p Perm) error {
	if !manager.Has(p) {
		return fmt.Errorf("%w: %s", ErrCannotManage, p)
	}

	return nil
}

// Unmanageable returns the permissions in a change that the manager may not
// make, so an interface can report all of them at once instead of one per
// attempt.
func Unmanageable(manager, current, next Set) []Perm {
	var out []Perm

	for _, p := range current.Diff(next) {
		if !manager.Has(p) {
			out = append(out, p)
		}
	}

	return out
}
