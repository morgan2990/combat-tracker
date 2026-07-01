package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"combatapp/auth"
	"combatapp/room"
	"combatapp/store"
	"golang.org/x/crypto/bcrypt"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Edition string `json:"edition"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint — empty body is fine
	edition := body.Edition
	if edition != "5e" && edition != "5.5e" {
		edition = "5e"
	}
	roomID := room.Global.CreateRoom(edition, userID)
	if rm, ok := room.Global.GetRoom(roomID); ok {
		go rm.PersistNow(&store.GlobalRooms)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"room_id": roomID,
		"edition": edition,
	})
}

func GetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")
	if roomID == "" {
		http.Error(w, "room_id required", http.StatusBadRequest)
		return
	}
	rm, found := room.Global.GetOrRestoreRoom(roomID, &store.GlobalRooms)
	if !found {
		http.NotFound(w, r)
		return
	}
	id, edition, isCombatActive := rm.Summary()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"room_id":          id,
		"edition":          edition,
		"is_combat_active": isCombatActive,
	})
}

// --- Auth ---

type authPayload struct {
	Username   string `json:"username"`
	Passphrase string `json:"passphrase"`
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	var body authPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Username == "" || body.Passphrase == "" {
		http.Error(w, "username and passphrase required", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Passphrase), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "could not hash passphrase", http.StatusInternalServerError)
		return
	}
	user, err := store.GlobalUsers.CreateUser(body.Username, string(hash))
	if err != nil {
		http.Error(w, "username taken", http.StatusConflict)
		return
	}
	sess, err := store.GlobalUsers.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sess.Token)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"username": user.Username})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var body authPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	user, err := store.GlobalUsers.GetUserByUsername(body.Username)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PassphraseHash), []byte(body.Passphrase)) != nil {
		http.Error(w, "invalid username or passphrase", http.StatusUnauthorized)
		return
	}
	sess, err := store.GlobalUsers.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sess.Token)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"username": user.Username})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("combatapp_session"); err == nil && c.Value != "" {
		store.GlobalUsers.DeleteSession(c.Value)
	}
	auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func Me(w http.ResponseWriter, r *http.Request) {
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
	pcs, err := store.Global.ListPCsByOwner(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pcs == nil {
		pcs = []store.PC{}
	}
	rooms, err := store.GlobalRooms.ListByOwner(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if rooms == nil {
		rooms = []store.RoomSummary{}
	}
	memberships, err := store.GlobalMemberships.ListByUser(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if memberships == nil {
		memberships = []store.RoomMembership{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user":         map[string]string{"username": user.Username, "display_name": user.DisplayName},
		"rooms":        rooms,
		"pcs":          pcs,
		"recent_rooms": memberships,
	})
}

// --- PCs ---

type pcPayload struct {
	Name             string `json:"name"`
	MaxHP            int    `json:"max_hp"`
	SharesInitiative bool   `json:"shares_initiative"`
}

func CreatePC(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body pcPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	pc, err := store.Global.CreatePC(userID, body.Name, body.MaxHP)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pc)
}

func UpdatePC(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	existing, err := store.Global.GetPCByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	if existing.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body pcPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	if err := store.Global.UpdatePC(id, body.Name, body.MaxHP); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type pcResponse struct {
	PC         *store.PC  `json:"pc"`
	Companions []store.PC `json:"companions"`
}

func GetPC(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	pc, err := store.Global.GetPCByID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if pc == nil {
		http.NotFound(w, r)
		return
	}
	if pc.OwnerUserID != userID {
		http.NotFound(w, r)
		return
	}
	companions, err := store.Global.GetCompanionsByParentID(id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if companions == nil {
		companions = []store.PC{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pcResponse{PC: pc, Companions: companions})
}

func CreateCompanion(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ResolveUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	parentID := r.PathValue("id")
	parent, err := store.Global.GetPCByID(parentID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if parent == nil || parent.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body pcPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MaxHP <= 0 {
		http.Error(w, "name and max_hp required", http.StatusBadRequest)
		return
	}
	companion, err := store.Global.CreateCompanion(parentID, userID, body.Name, body.MaxHP, body.SharesInitiative)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companion)
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
		m := store.Monster{
			Name:       name,
			Edition:    edition,
			MaxHP:      maxHP,
			IsCustom:   true,
			SourceType: "pdf",
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
		if err := store.UploadPDF(name, file, -1); err != nil {
			http.Error(w, "storage error: "+err.Error(), http.StatusBadGateway)
			return
		}
		m.PDFObjectKey = "monsters/" + name + ".pdf"
		saved, _, err := store.GlobalMonsters.UpsertMonster(m)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(saved)
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
	if m.Edition != "5e" && m.Edition != "5.5e" {
		http.Error(w, "edition must be \"5e\" or \"5.5e\"", http.StatusBadRequest)
		return
	}
	m.IsCustom = true
	saved, _, err := store.GlobalMonsters.UpsertMonster(m)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
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

func SearchMonsters(w http.ResponseWriter, r *http.Request) {
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
	hits, err := store.GlobalMonsters.SearchMonsters(q, edition)
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
