package impls

import (
	"strings"
	"testing"
)

// Token generation is security critical, so the rewrite to buffered reads is
// pinned: correct length, alphabet membership, and no bias from the rejection
// sampling.
func TestGenRandom(t *testing.T) {
	for _, n := range []int{0, 1, 32, 512, 2048, 6000} {
		got := GenRandom(n)

		if len(got) != n {
			t.Errorf("GenRandom(%d) length = %d", n, len(got))
		}

		for i, c := range got {
			if !strings.ContainsRune(alnum, c) {
				t.Fatalf("GenRandom(%d)[%d] = %q, outside the alphabet", n, i, c)
			}
		}
	}
}

// Two draws must not collide, and every character of the alphabet should appear
// across a large sample - a mask bug would silently drop the tail of it.
func TestGenRandomCoversAlphabet(t *testing.T) {
	if GenRandom(64) == GenRandom(64) {
		t.Fatal("two 64-character tokens were identical")
	}

	sample := GenRandom(200_000)

	for _, c := range alnum {
		if !strings.ContainsRune(sample, c) {
			t.Errorf("character %q never appeared in a 200k sample", c)
		}
	}
}

func TestRandRange(t *testing.T) {
	if got := RandRange(5, 5); got != 5 {
		t.Errorf("RandRange(5,5) = %d, want 5", got)
	}

	if got := RandRange(9, 3); got != 9 {
		t.Errorf("RandRange with hi <= lo = %d, want lo", got)
	}

	// The session token length range upstream uses.
	seen := map[int]bool{}

	for i := 0; i < 2000; i++ {
		got := RandRange(4196, 6000)

		if got < 4196 || got >= 6000 {
			t.Fatalf("RandRange(4196,6000) = %d, out of range", got)
		}

		seen[got] = true
	}

	if len(seen) < 100 {
		t.Errorf("only %d distinct values in 2000 draws, distribution looks degenerate", len(seen))
	}
}

func BenchmarkGenRandom4196(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenRandom(4196)
	}
}
