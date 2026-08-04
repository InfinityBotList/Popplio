package perms

import (
	"math"

	"popplio/state"
)

// OwnerRank is the rank of an instance owner.
//
// It is more senior than any staff role can ever be, since role indexes are
// constrained to be positive, so every "is the caller senior enough" check
// passes for an owner without that check needing to know about owners at all.
const OwnerRank int32 = math.MinInt32

// IsConfigOwner reports whether a user is one of the instance owners listed in
// the config's `arcadia.owners`.
//
// Owners are the operators of the instance itself rather than staff members:
// they are named in the file that starts the process, so anyone who can change
// the list can already change anything. Treating them as fully privileged is
// therefore a statement of fact, not a grant — and it is what stops the
// permission system from being able to lock its own operators out.
//
// An owner bypasses every staff permission check and every rank check. They do
// NOT bypass entity permissions: owning the instance says nothing about who may
// edit a particular user's bot, and that decision stays with the team.
func IsConfigOwner(userID string) bool {
	if state.Config == nil {
		return false
	}

	for _, owner := range state.Config.Arcadia.Owners {
		if owner.String() == userID {
			return true
		}
	}

	return false
}
