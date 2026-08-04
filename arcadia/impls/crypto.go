package impls

import (
	"crypto/rand"
	"encoding/binary"
)

const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// alnumMask is the smallest mask covering len(alnum)-1 (61), i.e. 6 bits. Any
// draw above the alphabet length is rejected and redrawn, which keeps the
// distribution uniform - taking a modulo would bias the first two characters.
const alnumMask = 1<<6 - 1

// GenRandom returns a random alphanumeric string of length n.
//
// HARDENING (§14b): the tokens this produces are panel session tokens and
// Popplio staff tokens, so this reads from crypto/rand. eureka's
// crypto.RandString would have been the obvious reuse, but it draws from
// math/rand seeded with the current time and its alphabet omits digits, which
// would both weaken and change the tokens Arcadia issues today.
//
// Randomness is drawn in one buffered read rather than per character. A login
// issues a 4196-6000 character session token plus a 2048 character Popplio
// token, so the naive form cost ~8000 reads and ~8000 big.Int allocations per
// login; this costs one read and one allocation in the common case.
func GenRandom(n int) string {
	if n <= 0 {
		return ""
	}

	out := make([]byte, n)

	// 64 bytes of slack covers the expected rejections (61/64 acceptance) without
	// a second read for any realistic length.
	buf := make([]byte, n+64)
	fill(buf)

	for i, read := 0, 0; i < n; {
		if read == len(buf) {
			fill(buf)
			read = 0
		}

		c := buf[read] & alnumMask
		read++

		if int(c) >= len(alnum) {
			continue
		}

		out[i] = alnum[c]
		i++
	}

	return string(out)
}

// RandRange returns a random integer in [lo, hi), matching Rust's
// rand::Rng::gen_range semantics (inclusive lower bound, exclusive upper).
func RandRange(lo, hi int) int {
	if hi <= lo {
		return lo
	}

	span := uint64(hi - lo)

	var buf [8]byte

	// Rejection sampling against the largest multiple of span that fits, so the
	// result is uniform rather than modulo-biased.
	limit := ^uint64(0) - (^uint64(0) % span) - 1

	for {
		fill(buf[:])

		v := binary.BigEndian.Uint64(buf[:])

		if v <= limit {
			return lo + int(v%span)
		}
	}
}

// fill reads from crypto/rand, treating failure as fatal: issuing a predictable
// session token would be worse than crashing.
func fill(buf []byte) {
	if _, err := rand.Read(buf); err != nil {
		panic("arcadia: crypto/rand unavailable: " + err.Error())
	}
}
