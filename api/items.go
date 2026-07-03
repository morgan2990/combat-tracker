package api

import (
	"net/http"

	"combatapp/store"
)

func SearchItems(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUser(w, r); !ok {
		return
	}
	q := r.URL.Query().Get("q")
	edition := r.URL.Query().Get("edition")
	if q == "" || edition == "" {
		http.Error(w, "q and edition are required", http.StatusBadRequest)
		return
	}
	if !requireValidEdition(w, edition) {
		return
	}
	hits, err := store.GlobalCatalogItems.SearchItems(q, edition)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if hits == nil {
		hits = []store.CatalogItemHit{}
	}
	writeJSON(w, http.StatusOK, hits)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUser(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	ci, err := store.GlobalCatalogItems.GetItemByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if ci == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, ci)
}
