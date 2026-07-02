package api

import (
	"encoding/json"
	"io"
	"net/http"

	"combatapp/auth"
	"combatapp/store"
)

// GetMonster and StreamMonsterPDF (below) resolve official, scrubber-imported
// monsters only, by name — official names are globally unique. DM-authored
// monsters are handled by the custom-monster handlers, keyed by id, since
// custom names are not unique across owners.

func GetMonster(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	m, err := store.GlobalMonsters.GetMonsterByName(name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if m == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func SearchMonsters(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	q := r.URL.Query().Get("q")
	edition := r.URL.Query().Get("edition")
	if q == "" || edition == "" {
		http.Error(w, "q and edition are required", http.StatusBadRequest)
		return
	}
	if edition != "5e" && edition != "5.5e" {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return
	}
	hits, err := store.GlobalMonsters.SearchMonsters(q, edition, userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if hits == nil {
		hits = []store.MonsterHit{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hits)
}

func StreamMonsterPDF(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	m, err := store.GlobalMonsters.GetMonsterByName(name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if m == nil || m.PDFObjectKey == "" {
		http.NotFound(w, r)
		return
	}
	rc, err := store.StreamPDF(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, rc)
}
