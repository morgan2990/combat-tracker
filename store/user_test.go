package store

import (
	"encoding/hex"
	"testing"
)

func TestNewID(t *testing.T) {
	id := NewID(8)
	if len(id) != 16 {
		t.Errorf("NewID(8) length = %d, want 16 (hex-encoded 8 bytes)", len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Errorf("NewID(8) = %q, not valid hex: %v", id, err)
	}

	if a, b := NewID(8), NewID(8); a == b {
		t.Errorf("NewID(8) returned the same value twice: %q", a)
	}
}
