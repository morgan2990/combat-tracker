package room

import (
	"math"
	"testing"
)

// TestRollD20Distribution verifies that rollD20 produces a uniform distribution
// over [1, 20] using a chi-squared goodness-of-fit test.
//
// With 20,000 rolls and 20 buckets the expected count per bucket is 1,000.
// We reject at p < 0.001 (chi-sq critical value ≈ 45.3 for 19 df).
func TestRollD20Distribution(t *testing.T) {
	const rolls = 20_000
	counts := make([]int, 21) // index 1-20 used

	for range rolls {
		v := rollD20()
		if v < 1 || v > 20 {
			t.Fatalf("rollD20() returned %d, want [1, 20]", v)
		}
		counts[v]++
	}

	expected := float64(rolls) / 20.0 // 1000.0

	chiSq := 0.0
	for face := 1; face <= 20; face++ {
		diff := float64(counts[face]) - expected
		chiSq += (diff * diff) / expected
	}

	// chi-squared critical value for df=19 at p=0.001 is ~45.3
	const criticalValue = 45.3
	if chiSq > criticalValue {
		t.Errorf("chi-squared = %.2f exceeds critical value %.2f (df=19, p=0.001) — distribution is not uniform", chiSq, criticalValue)
		t.Logf("Face distribution (expected %.0f each):", expected)
		for face := 1; face <= 20; face++ {
			bar := ""
			for range counts[face] / 20 {
				bar += "█"
			}
			t.Logf("  %2d: %4d  %s", face, counts[face], bar)
		}
	} else {
		t.Logf("chi-squared = %.2f (critical = %.2f, df=19, p=0.001) — distribution OK", chiSq, criticalValue)
	}

	// Sanity-check the range boundaries.
	if counts[1] == 0 || counts[20] == 0 {
		t.Errorf("boundary faces (1 or 20) never rolled in %d attempts — likely off-by-one in rollD20", rolls)
	}

	// Verify no face was rolled more than 3 standard deviations from expected
	// (σ = sqrt(expected * (1 - 1/20)) ≈ 30.8, so 3σ ≈ 92 deviation tolerated).
	sigma := math.Sqrt(expected * (1 - 1.0/20))
	for face := 1; face <= 20; face++ {
		if math.Abs(float64(counts[face])-expected) > 3*sigma*2 {
			t.Logf("  face %d: count=%d (expected=%.0f) — unusually skewed", face, counts[face], expected)
		}
	}
}

func TestRollD20Bounds(t *testing.T) {
	// Run 1000 rolls and assert every result is in [1, 20].
	for i := range 1000 {
		v := rollD20()
		if v < 1 || v > 20 {
			t.Errorf("roll %d: rollD20() = %d, want value in [1, 20]", i, v)
		}
	}
}

func TestActiveEntityID(t *testing.T) {
	r := &Room{State: RoomState{
		IsStarted:   true,
		ActiveIndex: 1,
		Entities: []Entity{
			{ID: "e1"},
			{ID: "e2"},
		},
	}}
	id := r.activeEntityID()
	if id == nil || *id != "e2" {
		t.Fatalf("activeEntityID() = %v, want pointer to \"e2\"", id)
	}

	r.State.IsStarted = false
	if id := r.activeEntityID(); id != nil {
		t.Errorf("activeEntityID() with IsStarted=false = %v, want nil", *id)
	}
}

func TestResolveActiveIndex(t *testing.T) {
	r := &Room{State: RoomState{
		Entities: []Entity{
			{ID: "e1"},
			{ID: "e2"},
			{ID: "e3"},
		},
	}}

	if idx := r.resolveActiveIndex(nil); idx != 0 {
		t.Errorf("resolveActiveIndex(nil) = %d, want 0", idx)
	}

	target := "e3"
	if idx := r.resolveActiveIndex(&target); idx != 2 {
		t.Errorf("resolveActiveIndex(%q) = %d, want 2", target, idx)
	}

	missing := "does-not-exist"
	if idx := r.resolveActiveIndex(&missing); idx != 0 {
		t.Errorf("resolveActiveIndex(%q) = %d, want fallback 0", missing, idx)
	}
}

func TestSnapshotConnectedStatus(t *testing.T) {
	r := &Room{
		State: RoomState{
			Entities: []Entity{
				{ID: "p1", Type: "player", SessionID: "sess-online"},
				{ID: "p2", Type: "player", SessionID: "sess-offline"},
				{ID: "c1", Type: "companion", OwnerID: "p1"},
				{ID: "k1", Type: "creature"},
			},
		},
		Clients: map[string]*Client{
			"sess-online": {Role: "player", Name: "Aragorn"},
		},
	}

	snap := r.snapshot()
	connected := make(map[string]bool, len(snap.Entities))
	for _, e := range snap.Entities {
		connected[e.ID] = e.Connected
	}

	if !connected["p1"] {
		t.Error("p1 (live session) should be connected")
	}
	if connected["p2"] {
		t.Error("p2 (no live session) should not be connected")
	}
	if connected["c1"] {
		t.Error("companion with no session_id should not be connected")
	}
	if connected["k1"] {
		t.Error("creature with no session_id should not be connected")
	}
}
