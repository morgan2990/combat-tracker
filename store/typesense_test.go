package store

import "testing"

// These cover the nil-client early-return branches shared by every
// Typesense-facing function: globalTypesense.client is nil whenever
// InitTypesense() hasn't been called (or failed, e.g. Typesense
// unreachable), which is the default and expected state in this package's
// plain unit tests. The real network-calling paths (schema creation,
// upsert/delete/search against a live server) are covered by
// typesense_integration_test.go against a real Typesense container.

func TestUpsertTypesenseDoc_NilClient_NoOp(t *testing.T) {
	if globalTypesense.client != nil {
		t.Fatal("expected globalTypesense.client to be nil in a plain unit test")
	}
	// Must not panic despite no client configured.
	upsertTypesenseDoc(typesenseMonsterDoc{ID: "1", Name: "Goblin"})
}

func TestDeleteTypesenseDoc_NilClient_NoOp(t *testing.T) {
	deleteTypesenseDoc("some-id")
}

func TestSearchTypesenseMonsters_NilClient_ReturnsEmpty(t *testing.T) {
	hits, err := searchTypesenseMonsters("Goblin", "5e", "user1")
	if err != nil {
		t.Fatalf("searchTypesenseMonsters() error = %v, want nil", err)
	}
	if len(hits) != 0 {
		t.Errorf("searchTypesenseMonsters() = %v, want empty", hits)
	}
}

func TestSyncItemToTypesense_NilClient_NoOp(t *testing.T) {
	syncItemToTypesense(CatalogItem{Name: "Longsword"})
}

func TestSearchTypesenseItems_NilClient_ReturnsEmpty(t *testing.T) {
	hits, err := searchTypesenseItems("Longsword", "5e")
	if err != nil {
		t.Fatalf("searchTypesenseItems() error = %v, want nil", err)
	}
	if len(hits) != 0 {
		t.Errorf("searchTypesenseItems() = %v, want empty", hits)
	}
}

func TestSyncMonsterToTypesense_NilClient_NoOp(t *testing.T) {
	// syncMonsterToTypesense/syncCustomMonsterToTypesense build the doc and
	// delegate to upsertTypesenseDoc, which no-ops on a nil client — just
	// confirm the call chain doesn't panic on a zero-value Monster.
	syncMonsterToTypesense(Monster{Name: "Goblin", Edition: "5e", MaxHP: 7})
}

func TestSyncCustomMonsterToTypesense_NilClient_NoOp(t *testing.T) {
	syncCustomMonsterToTypesense(CustomMonster{Name: "Bandit", Edition: "5e", MaxHP: 25, OwnerID: "owner1"})
}
