package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"combatapp/auth"
	"combatapp/store"
)

func CreateCustomMonster(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := store.GlobalUsers.GetUserByID(userID)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			if err.Error() == "http: request body too large" {
				http.Error(w, "file too large (max 20 MB)", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		edition := r.FormValue("edition")
		maxHP := 0
		if v := r.FormValue("max_hp"); v != "" {
			maxHP, _ = strconv.Atoi(v)
		}
		if name == "" || maxHP <= 0 {
			http.Error(w, "name and max_hp required", http.StatusBadRequest)
			return
		}
		if edition != "5e" && edition != "5.5e" {
			http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
			return
		}
		id := store.NewID()
		m := store.CustomMonster{
			ID:               id,
			Name:             name,
			Edition:          edition,
			MaxHP:            maxHP,
			SourceType:       "pdf",
			Private:          r.FormValue("private") == "true",
			OwnerID:          userID,
			OwnerDisplayName: user.DisplayName,
		}
		if v := r.FormValue("initiative_modifier"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				m.InitiativeModifier = &val
			}
		}
		file, _, err := r.FormFile("pdf")
		if err != nil {
			http.Error(w, "pdf file required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		if err := store.UploadCustomMonsterPDF(id, file, -1); err != nil {
			http.Error(w, "storage error: "+err.Error(), http.StatusBadGateway)
			return
		}
		m.PDFObjectKey = "custom-monsters/" + id + ".pdf"
		saved, err := store.GlobalCustomMonsters.CreateCustomMonster(m)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(saved)
		return
	}

	var m store.CustomMonster
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if m.Name == "" || m.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	if m.Edition != "5e" && m.Edition != "5.5e" {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return
	}
	m.OwnerID = userID
	m.OwnerDisplayName = user.DisplayName
	saved, err := store.GlobalCustomMonsters.CreateCustomMonster(m)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func GetCustomMonster(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	m, err := store.GlobalCustomMonsters.GetCustomMonsterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if m == nil {
		http.NotFound(w, r)
		return
	}
	if m.Private && m.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func UpdateCustomMonster(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.GlobalCustomMonsters.GetCustomMonsterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	if existing.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body store.CustomMonster
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	if body.Edition != "5e" && body.Edition != "5.5e" {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return
	}
	body.OwnerID = existing.OwnerID
	body.OwnerDisplayName = existing.OwnerDisplayName
	body.PDFObjectKey = existing.PDFObjectKey
	saved, err := store.GlobalCustomMonsters.UpdateCustomMonster(id, body)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func DeleteCustomMonster(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.GlobalCustomMonsters.GetCustomMonsterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	if existing.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := store.GlobalCustomMonsters.DeleteCustomMonster(id); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ListMyCustomMonsters(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	edition := r.URL.Query().Get("edition")
	monsters, err := store.GlobalCustomMonsters.ListCustomMonstersByOwner(userID, edition)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if monsters == nil {
		monsters = []store.CustomMonster{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monsters)
}

func StreamCustomMonsterPDF(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	m, err := store.GlobalCustomMonsters.GetCustomMonsterByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if m == nil || m.PDFObjectKey == "" {
		http.NotFound(w, r)
		return
	}
	if m.Private && m.OwnerID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	rc, err := store.StreamCustomMonsterPDF(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, rc)
}
