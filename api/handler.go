package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"combatapp/room"
	"combatapp/store"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	roomID, dmToken := room.Global.CreateRoom()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"room_id":  roomID,
		"dm_token": dmToken,
	})
}

func UpsertEntity(w http.ResponseWriter, r *http.Request) {
	var p store.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if p.Name == "" || p.MaxHP <= 0 || (p.Type != "player" && p.Type != "companion") {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if p.Type == "companion" && p.ParentPCName == "" {
		http.Error(w, "companion requires parent_pc_name", http.StatusBadRequest)
		return
	}
	if err := store.Global.UpsertEntity(p); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func UpsertMonster(w http.ResponseWriter, r *http.Request) {
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
		maxHP := 0
		if v := r.FormValue("max_hp"); v != "" {
			maxHP, _ = strconv.Atoi(v)
		}
		if name == "" || maxHP <= 0 {
			http.Error(w, "name and max_hp required", http.StatusBadRequest)
			return
		}
		m := store.Monster{
			Name:       name,
			MaxHP:      maxHP,
			SourceType: "pdf",
		}
		file, _, err := r.FormFile("pdf")
		if err != nil {
			http.Error(w, "pdf file required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		if err := store.UploadPDF(name, file, -1); err != nil {
			http.Error(w, "storage error: "+err.Error(), http.StatusBadGateway)
			return
		}
		m.PDFObjectKey = "monsters/" + name + ".pdf"
		if err := store.GlobalMonsters.UpsertMonster(m); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)
		return
	}

	var m store.Monster
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if m.Name == "" || m.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	if err := store.GlobalMonsters.UpsertMonster(m); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

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

type profileResponse struct {
	Profile    *store.Profile  `json:"profile"`
	Companions []store.Profile `json:"companions"`
}

func GetEntity(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	profile, err := store.Global.GetEntityByName(name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if profile == nil {
		http.NotFound(w, r)
		return
	}
	companions, err := store.Global.GetCompanionsByParent(name)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if companions == nil {
		companions = []store.Profile{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResponse{Profile: profile, Companions: companions})
}
