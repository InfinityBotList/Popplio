package impls

import (
	"crypto/rand"
	"math/big"
)

const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenRandom returns a random alphanumeric string of length n.
//
// HARDENING (§14b): the tokens this produces are panel session tokens and
// Popplio staff tokens, so this reads from crypto/rand. eureka's
// crypto.RandString would have been the obvious reuse, but it draws from
// math/rand seeded with the current time and its alphabet omits digits, which
// would both weaken and change the tokens Arcadia issues today.
func GenRandom(n int) string {
	out := make([]byte, n)
	max := big.NewInt(int64(len(alnum)))

	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand.Reader failing is not a recoverable condition; issuing a
			// predictable session token would be worse than crashing.
			panic("arcadia: crypto/rand unavailable: " + err.Error())
		}

		out[i] = alnum[idx.Int64()]
	}

	return string(out)
}

// RandRange returns a random integer in [lo, hi), matching Rust's
// rand::Rng::gen_range semantics (inclusive lower bound, exclusive upper).
func RandRange(lo, hi int) int {
	if hi <= lo {
		return lo
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	if err != nil {
		panic("arcadia: crypto/rand unavailable: " + err.Error())
	}

	return lo + int(n.Int64())
}
