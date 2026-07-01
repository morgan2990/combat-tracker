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
	Type             string   `json:"type"` // pc | creature | companion
	OwnerID          string   `json:"owner_id,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	// PCID is the Mongo PC/companion document this entity was instantiated from
	// (empty for creatures and for ad-hoc companions added via add_companion).
	// Used to match a reconnecting pc_id back to its entity, to resolve
	// refresh_from_profile lookups without relying on (no-longer-unique) Name,
	// and by the frontend to identify "which entity is mine" after connecting.
	PCID             string   `json:"pc_id,omitempty"`
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
	DisplayName         string   `json:"display_name,omitempty"`
	IsHidden            bool     `json:"is_hidden"`
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
	UserID    string // authenticated user id, resolved from the session cookie
	PCID      string // the PC this connection is playing, for role=player
	Name      string
	SessionID string
	MaxHP     int // resolved server-side from the owned PC; 0 means no PC loaded
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
	mu          sync.RWMutex
	State       RoomState
	OwnerUserID string
	Clients     map[string]*Client // session_id → *Client
	dirty       bool               // true if state has changed since the last persisted snapshot
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
		if *a == *b {
			iLair := r.State.Entities[i].Type == "lair_action"
			jLair := r.State.Entities[j].Type == "lair_action"
			if iLair != jLair {
				return jLair
			}
			return false
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

// isOwner returns true if the session exists and has the DM role. Ownership
// (session's UserID == Room.OwnerUserID) was already verified once, at
// connect time, in ValidateAndRegister — a session can only ever hold
// Role=="dm" if that check passed, so re-checking it here is unnecessary.
func (r *Room) isOwner(sessionID string) bool {
	c, ok := r.Clients[sessionID]
	return ok && c.Role == "dm"
}

// --- Combat turn flow ---

// StartCombat marks combat as started, sets round to 1, and resets active turn to index 0.
// Returns an error if any player or companion entity has a nil initiative.
func (r *Room) StartCombat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
		return errors.New("unauthorized")
	}
	if r.State.IsStarted {
		return errors.New("already started")
	}
	for _, e := range r.State.Entities {
		if (e.Type == "pc" || e.Type == "companion") && e.Initiative == nil {
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
	if !r.isOwner(sessionID) {
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
func (r *Room) AddCreature(sessionID, name string, maxHP int, initiativeModifier *int, quantity int, sourceType, referenceURL, pdfObjectKey, displayName string) error {
	if name == "" || maxHP <= 0 {
		return errors.New("invalid creature data")
	}
	if quantity < 1 {
		quantity = 1
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
		return errors.New("unauthorized")
	}
	r.appendCreatureGroup(name, maxHP, initiativeModifier, quantity, sourceType, referenceURL, pdfObjectKey, displayName)
	r.sortEntities()
	return nil
}

// appendCreatureGroup appends quantity instances of one creature group to
// r.State.Entities, with auto-numbered names/aliases and per-instance
// auto-rolled initiative when combat is active. Callers must hold r.mu and
// call r.sortEntities() themselves once all groups for the operation have
// been appended.
func (r *Room) appendCreatureGroup(name string, maxHP int, initiativeModifier *int, quantity int, sourceType, referenceURL, pdfObjectKey, displayName string) {
	for i := range quantity {
		entityName := name
		entityDisplayName := displayName
		if quantity > 1 {
			entityName = fmt.Sprintf("%s %d", name, i+1)
			if displayName != "" {
				entityDisplayName = fmt.Sprintf("%s %d", displayName, i+1)
			}
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
			DisplayName:        entityDisplayName,
		})
	}
}

// MonsterGroup is one already-resolved monster group ready to be appended to
// a room, as used by InjectEncounter. Resolution (looking up a monster by
// name or by custom-monster id, and skipping groups that no longer resolve)
// happens in the caller (ws/handler.go), which has access to the monster
// stores; room stays free of any Mongo-lookup concerns.
type MonsterGroup struct {
	Name               string
	MaxHP              int
	InitiativeModifier *int
	Quantity           int
	SourceType         string
	ReferenceURL       string
	PDFObjectKey       string
	DisplayName        string
}

// InjectEncounter appends every group in groups to the room in a single
// locked operation, then sorts and lets the caller broadcast once for the
// whole batch — equivalent to calling AddCreature once per group, but
// without the intermediate re-locks and re-sorts that would cause.
func (r *Room) InjectEncounter(sessionID string, groups []MonsterGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
		return errors.New("unauthorized")
	}
	for _, g := range groups {
		r.appendCreatureGroup(g.Name, g.MaxHP, g.InitiativeModifier, g.Quantity, g.SourceType, g.ReferenceURL, g.PDFObjectKey, g.DisplayName)
	}
	r.sortEntities()
	return nil
}

// RemoveEntity hard-deletes an entity and adjusts active_index to remain coherent.
func (r *Room) RemoveEntity(sessionID, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
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

// ToggleEntityVisibility flips the IsHidden flag on the given entity.
func (r *Room) ToggleEntityVisibility(sessionID, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
		return errors.New("unauthorized")
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.ID != entityID {
			continue
		}
		e.IsHidden = !e.IsHidden
		return nil
	}
	return errors.New("entity not found")
}

// RemoveDeadCreatures removes all entities with dead==true and type=="creature".
// Returns a non-nil error when nothing was removed (signals dispatcher to skip broadcast).
func (r *Room) RemoveDeadCreatures(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
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

// AddLairAction appends a fixed-initiative-20, HP-less "Lair Action" entity.
// It defaults to hidden so players don't learn a lair action exists until the
// DM reveals it via toggle_entity_visibility.
func (r *Room) AddLairAction(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
		return errors.New("unauthorized")
	}
	initiative := 20
	r.State.Entities = append(r.State.Entities, Entity{
		ID:         newToken(8),
		Name:       "Lair Action",
		Type:       "lair_action",
		Initiative: &initiative,
		MaxHP:      0,
		CurrentHP:  0,
		Conditions: []string{},
		IsHidden:   true,
	})
	r.sortEntities()
	return nil
}

// EndCombat terminates the active encounter: removes all creature entities, retains PCs
// and companions whose owner PC entity still exists, and resets combat state fields.
func (r *Room) EndCombat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
		return errors.New("unauthorized")
	}
	if !r.State.IsStarted {
		return errors.New("combat not started")
	}

	// Pass 1: collect surviving PC IDs.
	pcIDs := make(map[string]bool)
	for _, e := range r.State.Entities {
		if e.Type == "pc" {
			pcIDs[e.ID] = true
		}
	}

	// Pass 2: keep PCs and companions with a live owner; discard everything else.
	survivors := make([]Entity, 0, len(r.State.Entities))
	for _, e := range r.State.Entities {
		switch e.Type {
		case "pc":
			survivors = append(survivors, e)
		case "companion":
			if pcIDs[e.OwnerID] {
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
// displayName is also applied only to creature-type entities, but unlike name,
// blank is a meaningful value: it clears the entity's alias.
// If initiative changes, sortEntities is called with active position preserved.
func (r *Room) DMUpdateEntity(sessionID, entityID, name string, currentHP, tempHP, initiative int, conditions []string, dead bool, displayName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isOwner(sessionID) {
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
		if e.Type == "creature" {
			e.DisplayName = displayName
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

// SetupCharacter creates a PC entity for the given session using max_hp resolved
// server-side (from the owned PC document) at connect time.
func (r *Room) SetupCharacter(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.Clients[sessionID]
	if !ok || c.Role != "player" {
		return errors.New("session not found or not a player")
	}
	if c.MaxHP <= 0 {
		return errors.New("no PC loaded for this session")
	}
	for _, e := range r.State.Entities {
		if e.SessionID == sessionID {
			return errors.New("character already set up")
		}
	}
	r.State.Entities = append(r.State.Entities, Entity{
		ID:         newToken(8),
		Name:       c.Name,
		Type:       "pc",
		OwnerID:    "",
		SessionID:  sessionID,
		PCID:       c.PCID,
		MaxHP:      c.MaxHP,
		CurrentHP:  c.MaxHP,
		Initiative: nil,
		Conditions: []string{},
		Dead:       false,
	})
	r.sortEntities()
	return nil
}

// InstantiateCompanionsFromPC creates a companion entity for each of the given
// stored companion documents, owned by the PC entity for sessionID. Called by
// the WS handler immediately after SetupCharacter succeeds, replacing the
// previous client-driven add_companion auto-load.
func (r *Room) InstantiateCompanionsFromPC(sessionID string, companions []store.PC) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var ownerID string
	for _, e := range r.State.Entities {
		if e.SessionID == sessionID && e.Type == "pc" {
			ownerID = e.ID
			break
		}
	}
	if ownerID == "" {
		return errors.New("PC entity not found; complete character setup first")
	}
	for _, cp := range companions {
		r.State.Entities = append(r.State.Entities, Entity{
			ID:               newToken(8),
			Name:             cp.Name,
			Type:             "companion",
			OwnerID:          ownerID,
			PCID:             cp.ID,
			MaxHP:            cp.MaxHP,
			CurrentHP:        cp.MaxHP,
			Initiative:       nil,
			SharesInitiative: cp.SharesInitiative,
			Conditions:       []string{},
			Dead:             false,
		})
	}
	if len(companions) > 0 {
		r.sortEntities()
	}
	return nil
}

// SetInitiative sets the PC entity's initiative and propagates to shared companions.
func (r *Room) SetInitiative(sessionID string, initiative int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var pcEntityID string
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.SessionID == sessionID && e.Type == "pc" {
			e.Initiative = &initiative
			pcEntityID = e.ID
			break
		}
	}
	if pcEntityID == "" {
		return errors.New("PC entity not found; complete character setup first")
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.Type == "companion" && e.OwnerID == pcEntityID && e.SharesInitiative {
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

// AddCompanion creates an ad-hoc companion entity owned by the given player session
// (the in-room "Add Summon/Pet" flow; not linked to any stored PC document).
// initiative may be nil (not yet set).
func (r *Room) AddCompanion(sessionID, name string, maxHP int, sharesInitiative bool, initiative *int) error {
	if maxHP <= 0 || name == "" {
		return errors.New("invalid companion data")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerID := ""
	for _, e := range r.State.Entities {
		if e.SessionID == sessionID && e.Type == "pc" {
			ownerID = e.ID
			break
		}
	}
	if ownerID == "" {
		return errors.New("PC entity not found; complete character setup first")
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

// RefreshFromProfile re-fetches the player's owned PC document (via the
// connection's stored PCID) and updates max_hp for the PC entity and all
// linked companion entities (matched by their own stored PCID) in this room.
func (r *Room) RefreshFromProfile(sessionID string, st *store.Store) error {
	r.mu.RLock()
	c, ok := r.Clients[sessionID]
	r.mu.RUnlock()
	if !ok || c.Role != "player" {
		return errors.New("session not found or not a player")
	}

	pc, err := st.GetPCByID(c.PCID)
	if err != nil {
		return err
	}
	if pc == nil {
		return errors.New("PC not found")
	}
	companions, err := st.GetCompanionsByParentID(c.PCID)
	if err != nil {
		return err
	}

	companionMaxHP := make(map[string]int, len(companions))
	for _, cp := range companions {
		companionMaxHP[cp.ID] = cp.MaxHP
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	var pcEntityID string
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.SessionID == sessionID && e.Type == "pc" {
			e.MaxHP = pc.MaxHP
			if e.CurrentHP > e.MaxHP {
				e.CurrentHP = e.MaxHP
			}
			pcEntityID = e.ID
			break
		}
	}
	if pcEntityID == "" {
		return errors.New("PC entity not in room")
	}
	for i := range r.State.Entities {
		e := &r.State.Entities[i]
		if e.Type == "companion" && e.OwnerID == pcEntityID && e.PCID != "" {
			if newMax, ok := companionMaxHP[e.PCID]; ok {
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
// For role=dm, userID must match the room's OwnerUserID (verified by the caller's
// earlier authentication step, but re-checked here as the source of truth).
// For role=player, pcID/name/maxHP are resolved server-side by the caller from
// the owned PC document before calling this.
func (r *Room) ValidateAndRegister(role, userID, pcID, name string, maxHP int, conn *websocket.Conn) (*Client, int, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if role == "dm" && userID != r.OwnerUserID {
		return nil, 4003, "not the room owner"
	}
	if role == "player" && r.isPCActive(pcID) {
		return nil, 4009, "PC already active in this room"
	}

	c := &Client{
		Conn:      conn,
		Role:      role,
		UserID:    userID,
		PCID:      pcID,
		Name:      name,
		SessionID: newToken(8),
		MaxHP:     maxHP,
	}
	r.Clients[c.SessionID] = c

	if role == "player" {
		for i := range r.State.Entities {
			if r.State.Entities[i].PCID == pcID && r.State.Entities[i].Type == "pc" {
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

func (r *Room) isPCActive(pcID string) bool {
	for _, c := range r.Clients {
		if c.Role == "player" && c.PCID == pcID {
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
		if e.Type == "pc" && e.SessionID != "" {
			_, connected = r.Clients[e.SessionID]
		}
		entities[i] = store.RoomEntitySnapshot{
			ID:                 e.ID,
			Name:               e.Name,
			Type:               e.Type,
			OwnerID:            e.OwnerID,
			PCID:               e.PCID,
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
			DisplayName:        e.DisplayName,
			IsHidden:           e.IsHidden,
			Connected:          connected,
		}
	}
	return store.RoomSnapshot{
		RoomID:             r.State.RoomID,
		OwnerUserID:        r.OwnerUserID,
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

// CreateRoom generates a unique room ID, registers the room as owned by ownerUserID, and returns the ID.
func (reg *Registry) CreateRoom(edition, ownerUserID string) (roomID string) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	for {
		roomID = newRoomID()
		if _, exists := reg.rooms[roomID]; !exists {
			break
		}
	}
	reg.rooms[roomID] = &Room{
		State:       RoomState{RoomID: roomID, Edition: edition, Entities: []Entity{}},
		OwnerUserID: ownerUserID,
		Clients:     make(map[string]*Client),
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
			PCID:               e.PCID,
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
			DisplayName:        e.DisplayName,
			IsHidden:           e.IsHidden,
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
		OwnerUserID: snap.OwnerUserID,
		Clients:     make(map[string]*Client),
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
