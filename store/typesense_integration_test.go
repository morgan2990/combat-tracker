//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/typesense/typesense-go/v2/typesense"
)

// withRealTypesense points globalTypesense at a real client connected to the
// container started in TestMain (see party_integration_test.go), ensuring
// both collections exist, and restores the original (nil) globalTypesense
// afterward so tests that rely on "no Typesense configured" behavior
// (TestMonsterStore_SearchMonsters_NoTypesenseReturnsEmpty) aren't affected.
func withRealTypesense(t *testing.T) {
	t.Helper()
	client := typesense.NewClient(
		typesense.WithServer(testTypesenseURL),
		typesense.WithAPIKey(testTypesenseAPIKey),
		typesense.WithConnectionTimeout(5*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ensureMonsterCollection(ctx, client); err != nil {
		t.Fatalf("ensureMonsterCollection() error = %v", err)
	}
	if err := ensureItemCollection(ctx, client); err != nil {
		t.Fatalf("ensureItemCollection() error = %v", err)
	}

	original := globalTypesense
	globalTypesense = typesenseStore{client: client}
	t.Cleanup(func() { globalTypesense = original })
}

// pollForHits retries fn until it returns a non-empty slice or the timeout
// elapses — Typesense indexing is near-instant but not synchronous with the
// upsert call, so a single immediate search can flake.
func pollForHits[T any](t *testing.T, fn func() ([]T, error)) []T {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var hits []T
	for time.Now().Before(deadline) {
		var err error
		hits, err = fn()
		if err != nil {
			t.Fatalf("search error = %v", err)
		}
		if len(hits) > 0 {
			return hits
		}
		time.Sleep(50 * time.Millisecond)
	}
	return hits
}

func TestInitTypesense_ConnectsAndEnsuresCollections(t *testing.T) {
	original := globalTypesense
	t.Cleanup(func() { globalTypesense = original })

	t.Setenv("TYPESENSE_URL", testTypesenseURL)
	t.Setenv("TYPESENSE_API_KEY", testTypesenseAPIKey)

	InitTypesense()

	if globalTypesense.client == nil {
		t.Fatal("InitTypesense() left globalTypesense.client nil, want a connected client")
	}
}

func TestInitTypesense_MissingConfig_NoOp(t *testing.T) {
	original := globalTypesense
	t.Cleanup(func() { globalTypesense = original })
	globalTypesense = typesenseStore{}

	t.Setenv("TYPESENSE_URL", "")
	t.Setenv("TYPESENSE_API_KEY", "")

	InitTypesense()

	if globalTypesense.client != nil {
		t.Error("InitTypesense() with no URL/API key should leave globalTypesense.client nil")
	}
}

func TestInitTypesense_Unreachable_NoOp(t *testing.T) {
	original := globalTypesense
	t.Cleanup(func() { globalTypesense = original })
	globalTypesense = typesenseStore{}

	originalRetries, originalDelay := typesenseInitRetries, typesenseInitRetryDelay
	typesenseInitRetries, typesenseInitRetryDelay = 2, time.Millisecond
	t.Cleanup(func() { typesenseInitRetries, typesenseInitRetryDelay = originalRetries, originalDelay })

	t.Setenv("TYPESENSE_URL", "http://127.0.0.1:1")
	t.Setenv("TYPESENSE_API_KEY", "unreachable")

	InitTypesense()

	if globalTypesense.client != nil {
		t.Error("InitTypesense() against an unreachable server should leave globalTypesense.client nil")
	}
}

func TestSyncMonsterToTypesense_ThenSearch(t *testing.T) {
	withRealTypesense(t)

	name := "SearchableGoblin-" + NewID(6)
	syncMonsterToTypesense(Monster{ID: bson.NewObjectID(), Name: name, Edition: "5e", MaxHP: 7})

	hits := pollForHits(t, func() ([]MonsterHit, error) {
		return searchTypesenseMonsters(name, "5e", "anyone")
	})
	if len(hits) != 1 || hits[0].Name != name || hits[0].IsCustom {
		t.Fatalf("searchTypesenseMonsters() = %+v, want one official hit named %q", hits, name)
	}
}

func TestSyncCustomMonsterToTypesense_PrivacyFilter(t *testing.T) {
	withRealTypesense(t)

	suffix := NewID(6)
	ownerID := "owner-" + suffix
	otherID := "other-" + suffix
	privateName := "SecretBandit-" + suffix
	publicName := "OpenBandit-" + suffix

	syncCustomMonsterToTypesense(CustomMonster{
		ID: "cm-priv-" + suffix, Name: privateName, Edition: "5e", MaxHP: 20,
		Private: true, OwnerID: ownerID,
	})
	syncCustomMonsterToTypesense(CustomMonster{
		ID: "cm-pub-" + suffix, Name: publicName, Edition: "5e", MaxHP: 20,
		Private: false, OwnerID: ownerID,
	})

	// Owner can find their own private monster.
	ownerHits := pollForHits(t, func() ([]MonsterHit, error) {
		return searchTypesenseMonsters(privateName, "5e", ownerID)
	})
	if len(ownerHits) != 1 {
		t.Errorf("owner search for private monster = %+v, want 1 hit", ownerHits)
	}

	// A different user cannot find that same private monster.
	otherHits, err := searchTypesenseMonsters(privateName, "5e", otherID)
	if err != nil {
		t.Fatalf("searchTypesenseMonsters() error = %v", err)
	}
	if len(otherHits) != 0 {
		t.Errorf("non-owner search for private monster = %+v, want no hits", otherHits)
	}

	// Anyone can find the public monster.
	publicHits := pollForHits(t, func() ([]MonsterHit, error) {
		return searchTypesenseMonsters(publicName, "5e", otherID)
	})
	if len(publicHits) != 1 || !publicHits[0].IsCustom {
		t.Errorf("public custom monster search = %+v, want one custom hit", publicHits)
	}
}

func TestDeleteTypesenseDoc_RemovesFromIndex(t *testing.T) {
	withRealTypesense(t)

	name := "DoomedGoblin-" + NewID(6)
	id := "monster-" + NewID(6)
	// Uses the raw upsert path directly (rather than syncMonsterToTypesense)
	// so the document id is known and can be targeted precisely for delete.
	upsertTypesenseDoc(typesenseMonsterDoc{ID: id, Name: name, Edition: "5e", MaxHP: 7})

	pollForHits(t, func() ([]MonsterHit, error) {
		return searchTypesenseMonsters(name, "5e", "anyone")
	})

	deleteTypesenseDoc(id)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		hits, err := searchTypesenseMonsters(name, "5e", "anyone")
		if err != nil {
			t.Fatalf("searchTypesenseMonsters() error = %v", err)
		}
		found := false
		for _, h := range hits {
			if h.ID == id {
				found = true
			}
		}
		if !found {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("deleteTypesenseDoc() did not remove the document from the index in time")
}

func TestSyncItemToTypesense_ThenSearch(t *testing.T) {
	withRealTypesense(t)

	name := "SearchableLongsword-" + NewID(6)
	syncItemToTypesense(CatalogItem{Name: name, Edition: "5e", Category: "Weapon", Rarity: "Common"})

	hits := pollForHits(t, func() ([]CatalogItemHit, error) {
		return searchTypesenseItems(name, "5e")
	})
	if len(hits) != 1 || hits[0].Name != name || hits[0].Category != "Weapon" {
		t.Fatalf("searchTypesenseItems() = %+v, want one hit named %q", hits, name)
	}
}

func TestSearchTypesenseMonsters_EditionMismatch_NoHits(t *testing.T) {
	withRealTypesense(t)

	name := "EditionScopedGoblin-" + NewID(6)
	syncMonsterToTypesense(Monster{ID: bson.NewObjectID(), Name: name, Edition: "5e", MaxHP: 7})
	pollForHits(t, func() ([]MonsterHit, error) {
		return searchTypesenseMonsters(name, "5e", "anyone")
	})

	hits, err := searchTypesenseMonsters(name, "3.5e", "anyone")
	if err != nil {
		t.Fatalf("searchTypesenseMonsters() error = %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("cross-edition search = %+v, want no hits", hits)
	}
}
