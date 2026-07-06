package store

import (
	"strings"
	"testing"
)

func TestNewPartyJoinCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := newPartyJoinCode()

		if len(code) != joinCodeLength {
			t.Fatalf("newPartyJoinCode() = %q, want length %d", code, joinCodeLength)
		}
		for _, c := range code {
			if !strings.ContainsRune(joinCodeChars, c) {
				t.Fatalf("newPartyJoinCode() = %q contains char %q not in joinCodeChars", code, c)
			}
		}
		seen[code] = true
	}

	// With 33^5 (~39M) possible codes, 100 draws colliding even once is
	// vanishingly unlikely — this is really a sanity check that the RNG
	// isn't returning a constant value.
	if len(seen) < 90 {
		t.Errorf("got only %d distinct codes out of 100 draws — RNG looks degenerate", len(seen))
	}
}

func TestJoinCodeCharsExcludesAmbiguousCharacters(t *testing.T) {
	for _, ambiguous := range []rune{'I', 'O', '0', '1'} {
		if strings.ContainsRune(joinCodeChars, ambiguous) {
			t.Errorf("joinCodeChars contains visually ambiguous character %q", ambiguous)
		}
	}
}
