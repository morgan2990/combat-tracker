package room

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"combatapp/store"
	"github.com/gorilla/websocket"
)

const idChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Entity struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"` // player | creature | companion
	OwnerID          string   `json:"owner_id,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	MaxHP            int      `json:"max_hp"`
	CurrentHP        int      `json:"current_hp"`
	TempHP           int      `json:"temp_hp"`
	Initiative       *int     `json:"initiative"`
	SharesInitiative bool     `json:"shares_initiative"`
	Conditions       []string `json:"conditions"`
	Dead             bool     `json:"dead"`
	SourceType       string   `json:"source_type,omitempty"`
	ReferenceURL     string   `json:"reference_url,omitempty"`
	PDFObjectKey        string   `json:"pdf_object_key,omitempty"`
	InitiativeModifier  *int     `json:"initiative_modifier,omitempty"`
	InitiativeRoll      *int     `json:"initiative_roll,omitempty"`
}

type RoomState struct {
	RoomID      string   `json:"room_id"`
	Edition     string   `json:"edition"`
	IsStarted   bool     `json:"is_started"`
	Round       int      `json:"round"`
	ActiveIndex int      `json:"active_index"`
	Entities    []Entity `json:"entities"`
}

// Client represents a connected WebSocket session.
type Client struct {
	mu        sync.Mutex
	Conn      *websocket.Conn
	Role      string // dm | player
	Name      string
	SessionID string
	MaxHP     int // populated from profile on join; 0 means no profile loaded
}

func (c *Client) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteJSON(v)
}

func (c *Client) WritePing() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// Room holds live combat state and all active WebSocket connections.
type Room struct {
	mu      sync.RWMutex
	State   RoomState
	DMToken string
	Clients map[string]*Client // session_id → *Client
	dirty   bool               // true if state has changed since the last persisted snapshot
}

// sortEntities always re-sorts State.Entities descending by initiative.
// When combat is live (is_started), the active entity's position is preserved by ID.
func (r *Room) sortEntities() {
	var activeID string
	if r.State.IsStarted && r.State.ActiveIndex >= 0 && r.State.ActiveIndex < len(r.State.Entities) {
		activeID = r.State.Entities[r.State.ActiveIndex].ID
	}

	sort.SliceStable(r.State.Entities, func(i, j int) bool {
		a, b := r.State.Entities[i].Initiative, r.State.Entities[j].Initiative
		if a == nil && b == nil {
			return false
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return *a > *b
	})

	if activeID != "" {
		for i, e := range r.State.Entities {
			if e.ID == activeID {
				r.State.ActiveIndex = i
				break
			}
		}
	}
}

// isDM returns true if the session exists and has the DM role.
func (r *Room) isDM(sessionID string) bool {
	c, ok := r.Clients[sessionID]
	return ok && c.Role == "dm"
}

// --- Combat turn flow ---

// StartCombat marks combat as started, sets round to 1, and resets active turn to index 0.
// Returns an error if any player or companion entity has a nil initiative.
func (r *Room) StartCombat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	if r.State.IsStarted {
		return errors.New("already started")
	}
	for _, e := range r.State.Entities {
		if (e.Type == "player" || e.Type == "companion") && e.Initiative == nil {
			return errors.New("all players and companions must have initiative set before starting combat")
		}
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.Type == "creature" && e.InitiativeModifier != nil && e.Initiative == nil {
			d := rollD20()
			total := d + *e.InitiativeModifier
			e.InitiativeRoll = &d
			e.Initiative = &total
		}
	}
	r.sortEntities()
	r.State.IsStarted = true
	r.State.Round = 1
	r.State.ActiveIndex = 0
	return nil
}

// NextTurn advances the active turn, wrapping at the end of the list and incrementing the round.
func (r *Room) NextTurn(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	if !r.State.IsStarted {
		return errors.New("combat not started")
	}
	n := len(r.State.Entities)
	if n == 0 {
		return nil
	}
	if r.State.ActiveIndex >= n-1 {
		r.State.ActiveIndex = 0
		r.State.Round++
	} else {
		r.State.ActiveIndex++
	}
	return nil
}

// --- Creature management ---

// AddCreature creates one or more creature entities and sorts them into the initiative order.
// When quantity > 1, each entity is named with an auto-number suffix (e.g. "Goblin 1").
// A single sort and broadcast is expected by the caller after this returns.
func (r *Room) AddCreature(sessionID, name string, maxHP int, initiativeModifier *int, quantity int, sourceType, referenceURL, pdfObjectKey string) error {
	if name == "" || maxHP <= 0 {
		return errors.New("invalid creature data")
	}
	if quantity < 1 {
		quantity = 1
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	for i := range quantity {
		entityName := name
		if quantity > 1 {
			entityName = fmt.Sprintf("%s %d", name, i+1)
		}
		var init *int
		var roll *int
		if r.State.IsStarted && initiativeModifier != nil {
			d := rollD20()
			total := d + *initiativeModifier
			roll = &d
			init = &total
		}
		r.State.Entities = append(r.State.Entities, Entity{
			ID:                 newToken(8),
			Name:               entityName,
			Type:               "creature",
			MaxHP:              maxHP,
			CurrentHP:          maxHP,
			Initiative:         init,
			InitiativeModifier: initiativeModifier,
			InitiativeRoll:     roll,
			Conditions:         []string{},
			Dead:               false,
			SourceType:         sourceType,
			ReferenceURL:       referenceURL,
			PDFObjectKey:       pdfObjectKey,
		})
	}
	r.sortEntities()
	return nil
}

// RemoveEntity hard-deletes an entity and adjusts active_index to remain coherent.
func (r *Room) RemoveEntity(sessionID, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	idx := -1
	for i, e := range r.State.Entities {
		if e.ID == entityID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("entity not found")
	}
	r.State.Entities = append(r.State.Entities[:idx], r.State.Entities[idx+1:]...)
	switch {
	case idx < r.State.ActiveIndex:
		r.State.ActiveIndex--
	case idx == r.State.ActiveIndex:
		if r.State.ActiveIndex >= len(r.State.Entities) {
			r.State.ActiveIndex = 0
		}
	}
	return nil
}

// RemoveDeadCreatures removes all entities with dead==true and type=="creature".
// Returns a non-nil error when nothing was removed (signals dispatcher to skip broadcast).
func (r *Room) RemoveDeadCreatures(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}

	var activeID string
	if r.State.IsStarted && r.State.ActiveIndex < len(r.State.Entities) {
		activeID = r.State.Entities[r.State.ActiveIndex].ID
	}

	filtered := make([]Entity, 0, len(r.State.Entities))
	for _, e := range r.State.Entities {
		if e.Dead && e.Type == "creature" {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == len(r.State.Entities) {
		return errors.New("nothing to remove")
	}
	r.State.Entities = filtered

	r.State.ActiveIndex = 0
	if activeID != "" {
		for i, e := range r.State.Entities {
			if e.ID == activeID {
				r.State.ActiveIndex = i
				break
			}
		}
	}
	return nil
}

// EndCombat terminates the active encounter: removes all creature entities, retains players
// and companions whose owner player entity still exists, and resets combat state fields.
func (r *Room) EndCombat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	if !r.State.IsStarted {
		return errors.New("combat not started")
	}

	// Pass 1: collect surviving player IDs.
	playerIDs := make(map[string]bool)
	for _, e := range r.State.Entities {
		if e.Type == "player" {
			playerIDs[e.ID] = true
		}
	}

	// Pass 2: keep players and companions with a live owner; discard everything else.
	survivors := make([]Entity, 0, len(r.State.Entities))
	for _, e := range r.State.Entities {
		switch e.Type {
		case "player":
			survivors = append(survivors, e)
		case "companion":
			if playerIDs[e.OwnerID] {
				survivors = append(survivors, e)
			}
		}
	}

	r.State.Entities = survivors
	r.State.IsStarted = false
	r.State.Round = 0
	r.State.ActiveIndex = 0
	return nil
}

// --- DM entity override ---

// DMUpdateEntity applies DM-level edits to any entity without ownership checks.
// The name field is applied only to creature-type entities.
// If initiative changes, sortEntities is called with active position preserved.
func (r *Room) DMUpdateEntity(sessionID, entityID, name string, currentHP, tempHP, initiative int, conditions []string, dead bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.ID != entityID {
			continue
		}
		if e.Type == "creature" && name != "" {
			e.Name = name
		}
		if currentHP > e.MaxHP {
			currentHP = e.MaxHP
		}
		if currentHP < 0 {
			currentHP = 0
		}
		if tempHP < 0 {
			tempHP = 0
		}
		if conditions == nil {
			conditions = []string{}
		}
		initiativeChanged := e.Initiative == nil || *e.Initiative != initiative
		e.CurrentHP = currentHP
		e.TempHP = tempHP
		e.Initiative = &initiative
		e.Conditions = conditions
		e.Dead = dead
		if initiativeChanged {
			r.sortEntities()
		}
		return nil
	}
	return errors.New("entity not found")
}

// --- Player-initiated methods (Epic 2) ---

// SetupCharacter creates a player entity for the given session using max_hp from the client's profile.
func (r *Room) SetupCharacter(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.Clients[sessionID]
	if !ok || c.Role != "player" {
		return errors.New("session not found or not a player")
	}
	if c.MaxHP <= 0 {
		return errors.New("no profile loaded for this session")
	}
	for _, e := range r.State.Entities {
		if e.SessionID == sessionID {
			return errors.New("character already set up")
		}
	}
	r.State.Entities = append(r.State.Entities, Entity{
		ID:         newToken(8),
		Name:       c.Name,
		Type:       "player",
		SessionID:  sessionID,
		MaxHP:      c.MaxHP,
		CurrentHP:  c.MaxHP,
		Initiative: nil,
		Conditions: []string{},
		Dead:       false,
	})
	r.sortEntities()
	return nil
}

// SetInitiative sets the player entity's initiative and propagates to shared companions.
func (r *Room) SetInitiative(sessionID string, initiative int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var playerEntityID string
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.SessionID == sessionID && e.Type == "player" {
			e.Initiative = &initiative
			playerEntityID = e.ID
			break
		}
	}
	if playerEntityID == "" {
		return errors.New("player entity not found; complete character setup first")
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.Type == "companion" && e.OwnerID == playerEntityID && e.SharesInitiative {
			e.Initiative = &initiative
		}
	}
	r.sortEntities()
	return nil
}

// UpdateEntity applies HP and condition changes to an entity, enforcing ownership.
func (r *Room) UpdateEntity(sessionID, entityID string, currentHP, tempHP int, conditions []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	myEntityID := ""
	for _, e := range r.State.Entities {
		if e.SessionID == sessionID {
			myEntityID = e.ID
			break
		}
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.ID != entityID {
			continue
		}
		ownEntity := e.SessionID == sessionID
		ownCompanion := e.Type == "companion" && e.OwnerID == myEntityID && myEntityID != ""
		if !ownEntity && !ownCompanion {
			return errors.New("unauthorized")
		}
		if currentHP > e.MaxHP {
			currentHP = e.MaxHP
		}
		if currentHP < 0 {
			currentHP = 0
		}
		if tempHP < 0 {
			tempHP = 0
		}
		if conditions == nil {
			conditions = []string{}
		}
		e.CurrentHP = currentHP
		e.TempHP = tempHP
		e.Conditions = conditions
		return nil
	}
	return errors.New("entity not found")
}

// AddCompanion creates a companion entity owned by the given player session.
// initiative may be nil (not yet set).
func (r *Room) AddCompanion(sessionID, name string, maxHP int, sharesInitiative bool, initiative *int) error {
	if maxHP <= 0 || name == "" {
		return errors.New("invalid companion data")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerID := ""
	for _, e := range r.State.Entities {
		if e.SessionID == sessionID && e.Type == "player" {
			ownerID = e.ID
			break
		}
	}
	if ownerID == "" {
		return errors.New("player entity not found; complete character setup first")
	}
	r.State.Entities = append(r.State.Entities, Entity{
		ID:               newToken(8),
		Name:             name,
		Type:             "companion",
		OwnerID:          ownerID,
		MaxHP:            maxHP,
		CurrentHP:        maxHP,
		Initiative:       initiative,
		SharesInitiative: sharesInitiative,
		Conditions:       []string{},
		Dead:             false,
	})
	r.sortEntities()
	return nil
}

// RefreshFromProfile re-fetches the player's MongoDB profile and updates max_hp
// for the player entity and all linked companions in this room.
func (r *Room) RefreshFromProfile(sessionID string, st *store.Store) error {
	r.mu.RLock()
	c, ok := r.Clients[sessionID]
	r.mu.RUnlock()
	if !ok || c.Role != "player" {
		return errors.New("session not found or not a player")
	}

	profile, err := st.GetEntityByName(c.Name)
	if err != nil {
		return err
	}
	if profile == nil {
		return errors.New("profile not found")
	}
	companions, err := st.GetCompanionsByParent(c.Name)
	if err != nil {
		return err
	}

	companionMaxHP := make(map[string]int, len(companions))
	for _, cp := range companions {
		companionMaxHP[cp.Name] = cp.MaxHP
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	var playerEntityID string
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.SessionID == sessionID && e.Type == "player" {
			e.MaxHP = profile.MaxHP
			if e.CurrentHP > e.MaxHP {
				e.CurrentHP = e.MaxHP
			}
			playerEntityID = e.ID
			break
		}
	}
	if playerEntityID == "" {
		return errors.New("player entity not in room")
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.Type == "companion" && e.OwnerID == playerEntityID {
			if newMax, ok := companionMaxHP[e.Name]; ok {
				e.MaxHP = newMax
				if e.CurrentHP > e.MaxHP {
					e.CurrentHP = e.MaxHP
				}
			}
		}
	}
	return nil
}

// ValidateAndRegister validates a join attempt and registers the client.
// maxHP is used for player sessions to store the profile value; pass 0 for DM sessions.
func (r *Room) ValidateAndRegister(role, dmToken, name string, conn *websocket.Conn, maxHP int) (*Client, int, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if role == "dm" && dmToken != r.DMToken {
		return nil, 4003, "invalid dm token"
	}
	if role == "player" && r.isNameTaken(name) {
		return nil, 4009, "name already taken"
	}

	c := &Client{
		Conn:      conn,
		Role:      role,
		Name:      name,
		SessionID: newToken(8),
		MaxHP:     maxHP,
	}
	r.Clients[c.SessionID] = c

	if role == "player" {
		for i := range r.State.Entities {
			if r.State.Entities[i].Name == name && r.State.Entities[i].Type == "player" {
				r.State.Entities[i].SessionID = c.SessionID
				break
			}
		}
	}
	return c, 0, ""
}

// RemoveClient removes a session from the room's connection map.
func (r *Room) RemoveClient(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Clients, sessionID)
}

// BroadcastState sends the full RoomState JSON to every connected client.
func (r *Room) BroadcastState() {
	r.mu.RLock()
	state := r.State
	clients := make([]*Client, 0, len(r.Clients))
	for _, c := range r.Clients {
		clients = append(clients, c)
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.WriteJSON(state)
	}
}

// Summary returns the room's identifying metadata under a read lock, safe for
// callers outside the room package (e.g. REST handlers).
func (r *Room) Summary() (roomID, edition string, isCombatActive bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.State.RoomID, r.State.Edition, r.State.IsStarted
}

func (r *Room) isNameTaken(name string) bool {
	for _, c := range r.Clients {
		if c.Role == "player" && c.Name == name {
			return true
		}
	}
	return false
}

// --- Persistence ---

// MarkDirty flags the room as having unsaved changes, to be picked up by the
// next periodic sweep.
func (r *Room) MarkDirty() {
	r.mu.Lock()
	r.dirty = true
	r.mu.Unlock()
}

// activeEntityID returns the ID of the currently active entity, or nil if
// combat has not started. Callers must hold at least r.mu.RLock().
func (r *Room) activeEntityID() *string {
	if !r.State.IsStarted || r.State.ActiveIndex < 0 || r.State.ActiveIndex >= len(r.State.Entities) {
		return nil
	}
	id := r.State.Entities[r.State.ActiveIndex].ID
	return &id
}

// resolveActiveIndex finds the index of the entity matching id, falling back
// to 0 if id is nil or no entity matches.
func (r *Room) resolveActiveIndex(id *string) int {
	if id == nil {
		return 0
	}
	for i, e := range r.State.Entities {
		if e.ID == *id {
			return i
		}
	}
	return 0
}

// snapshot builds a persistable RoomSnapshot from the room's current state.
// Callers must hold at least r.mu.RLock().
func (r *Room) snapshot() store.RoomSnapshot {
	entities := make([]store.RoomEntitySnapshot, len(r.State.Entities))
	for i, e := range r.State.Entities {
		connected := false
		if e.Type == "player" && e.SessionID != "" {
			_, connected = r.Clients[e.SessionID]
		}
		entities[i] = store.RoomEntitySnapshot{
			ID:                 e.ID,
			Name:               e.Name,
			Type:               e.Type,
			OwnerID:            e.OwnerID,
			MaxHP:              e.MaxHP,
			CurrentHP:          e.CurrentHP,
			TempHP:             e.TempHP,
			Initiative:         e.Initiative,
			SharesInitiative:   e.SharesInitiative,
			Conditions:         e.Conditions,
			Dead:               e.Dead,
			SourceType:         e.SourceType,
			ReferenceURL:       e.ReferenceURL,
			PDFObjectKey:       e.PDFObjectKey,
			InitiativeModifier: e.InitiativeModifier,
			InitiativeRoll:     e.InitiativeRoll,
			Connected:          connected,
		}
	}
	return store.RoomSnapshot{
		RoomID:             r.State.RoomID,
		DMToken:            r.DMToken,
		IsCombatActive:     r.State.IsStarted,
		CurrentRound:       r.State.Round,
		ActiveTurnEntityID: r.activeEntityID(),
		Edition:            r.State.Edition,
		Entities:           entities,
	}
}

// PersistNow snapshots the room's current state, clears the dirty flag, and
// writes the snapshot to MongoDB. The MongoDB write happens outside the room
// lock, so this is safe to call from its own goroutine without blocking other
// room operations. If the write fails, the room is re-marked dirty so the
// next periodic sweep retries it.
func (r *Room) PersistNow(st *store.RoomStore) {
	r.mu.Lock()
	snap := r.snapshot()
	r.dirty = false
	r.mu.Unlock()

	if err := st.SaveRoomSnapshot(snap); err != nil {
		r.MarkDirty()
	}
}

// Registry is the global in-memory store of all active rooms.
type Registry struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

var Global = &Registry{rooms: make(map[string]*Room)}

// CreateRoom generates a unique room ID and DM token, registers the room, and returns both.
func (reg *Registry) CreateRoom(edition string) (roomID, dmToken string) {
	dmToken = newToken(4)
	reg.mu.Lock()
	defer reg.mu.Unlock()
	for {
		roomID = newRoomID()
		if _, exists := reg.rooms[roomID]; !exists {
			break
		}
	}
	reg.rooms[roomID] = &Room{
		State:   RoomState{RoomID: roomID, Edition: edition, Entities: []Entity{}},
		DMToken: dmToken,
		Clients: make(map[string]*Client),
	}
	return
}

// GetRoom retrieves a room by ID, checking only the in-memory registry.
func (reg *Registry) GetRoom(roomID string) (*Room, bool) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	rm, ok := reg.rooms[roomID]
	return rm, ok
}

// GetOrRestoreRoom retrieves a room by ID, checking the in-memory registry
// first and falling back to MongoDB if not found in memory. A room restored
// from MongoDB is registered into the registry before being returned.
func (reg *Registry) GetOrRestoreRoom(roomID string, st *store.RoomStore) (*Room, bool) {
	if rm, ok := reg.GetRoom(roomID); ok {
		return rm, true
	}

	snap, err := st.GetRoomSnapshot(roomID)
	if err != nil || snap == nil {
		return nil, false
	}
	rm := inflateRoom(*snap)

	reg.mu.Lock()
	defer reg.mu.Unlock()
	if existing, ok := reg.rooms[roomID]; ok {
		return existing, true
	}
	reg.rooms[roomID] = rm
	return rm, true
}

// SweepDirty persists every room that has been marked dirty since its last save.
func (reg *Registry) SweepDirty(st *store.RoomStore) {
	reg.mu.RLock()
	rooms := make([]*Room, 0, len(reg.rooms))
	for _, rm := range reg.rooms {
		rooms = append(rooms, rm)
	}
	reg.mu.RUnlock()

	for _, rm := range rooms {
		rm.mu.RLock()
		dirty := rm.dirty
		rm.mu.RUnlock()
		if dirty {
			rm.PersistNow(st)
		}
	}
}

// inflateRoom converts a persisted RoomSnapshot back into a live Room, with an
// empty Clients map since no WebSocket connection survives a process restart.
func inflateRoom(snap store.RoomSnapshot) *Room {
	entities := make([]Entity, len(snap.Entities))
	for i, e := range snap.Entities {
		entities[i] = Entity{
			ID:                 e.ID,
			Name:               e.Name,
			Type:               e.Type,
			OwnerID:            e.OwnerID,
			MaxHP:              e.MaxHP,
			CurrentHP:          e.CurrentHP,
			TempHP:             e.TempHP,
			Initiative:         e.Initiative,
			SharesInitiative:   e.SharesInitiative,
			Conditions:         e.Conditions,
			Dead:               e.Dead,
			SourceType:         e.SourceType,
			ReferenceURL:       e.ReferenceURL,
			PDFObjectKey:       e.PDFObjectKey,
			InitiativeModifier: e.InitiativeModifier,
			InitiativeRoll:     e.InitiativeRoll,
		}
	}
	rm := &Room{
		State: RoomState{
			RoomID:    snap.RoomID,
			Edition:   snap.Edition,
			IsStarted: snap.IsCombatActive,
			Round:     snap.CurrentRound,
			Entities:  entities,
		},
		DMToken: snap.DMToken,
		Clients: make(map[string]*Client),
	}
	rm.State.ActiveIndex = rm.resolveActiveIndex(snap.ActiveTurnEntityID)
	return rm
}

func newRoomID() string {
	b := make([]byte, 5)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(idChars))))
		b[i] = idChars[n.Int64()]
	}
	return string(b)
}

func newToken(byteLen int) string {
	b := make([]byte, byteLen)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func rollD20() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(20))
	return int(n.Int64()) + 1
}
