package store

import "testing"

func TestShouldPreserveCustom(t *testing.T) {
	cases := []struct {
		name                               string
		existingIsCustom, incomingIsCustom bool
		want                               bool
	}{
		{"non-custom write against existing custom doc is skipped", true, false, true},
		{"custom write always proceeds, even over a custom doc", true, true, false},
		{"custom write always proceeds, even over a non-custom doc", false, true, false},
		{"non-custom write proceeds when no conflicting custom doc", false, false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := shouldPreserveCustom(c.existingIsCustom, c.incomingIsCustom)
			if got != c.want {
				t.Errorf("shouldPreserveCustom(%v, %v) = %v, want %v", c.existingIsCustom, c.incomingIsCustom, got, c.want)
			}
		})
	}
}
