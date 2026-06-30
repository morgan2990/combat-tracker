package room

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
)

const idChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Entity struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"` // player | creature | companion
	OwnerID    string   `json:"owner_id,omitempty"`
	SessionID  string   `json:"session_id,omitempty"`
	MaxHP      int      `json:"max_hp"`
	CurrentHP  int      `json:"current_hp"`
	TempHP     int      `json:"temp_hp"`
	Initiative int      `json:"initiative"`
	Conditions []string `json:"conditions"`
	Dead       bool     `json:"dead"`
}

type RoomState struct {
	RoomID      string   `json:"room_id"`
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
}

// sortEntities always re-sorts State.Entities descending by initiative.
// When combat is live (is_started), the active entity's position is preserved by ID.
func (r *Room) sortEntities() {
	var activeID string
	if r.State.IsStarted && r.State.ActiveIndex >= 0 && r.State.ActiveIndex < len(r.State.Entities) {
		activeID = r.State.Entities[r.State.ActiveIndex].ID
	}

	sort.SliceStable(r.State.Entities, func(i, j int) bool {
		return r.State.Entities[i].Initiative > r.State.Entities[j].Initiative
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
func (r *Room) StartCombat(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	if r.State.IsStarted {
		return errors.New("already started")
	}
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

// AddCreature creates a creature entity and sorts it into the initiative order.
func (r *Room) AddCreature(sessionID, name string, maxHP, initiative int) error {
	if name == "" || maxHP <= 0 {
		return errors.New("invalid creature data")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isDM(sessionID) {
		return errors.New("unauthorized")
	}
	r.State.Entities = append(r.State.Entities, Entity{
		ID:         newToken(8),
		Name:       name,
		Type:       "creature",
		MaxHP:      maxHP,
		CurrentHP:  maxHP,
		Initiative: initiative,
		Conditions: []string{},
		Dead:       false,
	})
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
		initiativeChanged := initiative != e.Initiative
		e.CurrentHP = currentHP
		e.TempHP = tempHP
		e.Initiative = initiative
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

// SetupCharacter creates a player entity for the given session.
func (r *Room) SetupCharacter(sessionID string, maxHP, initiative int) error {
	if maxHP <= 0 {
		return errors.New("max_hp must be greater than 0")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.Clients[sessionID]
	if !ok || c.Role != "player" {
		return errors.New("session not found or not a player")
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
		MaxHP:      maxHP,
		CurrentHP:  maxHP,
		Initiative: initiative,
		Conditions: []string{},
		Dead:       false,
	})
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
func (r *Room) AddCompanion(sessionID, name string, maxHP, initiative int) error {
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
		ID:         newToken(8),
		Name:       name,
		Type:       "companion",
		OwnerID:    ownerID,
		MaxHP:      maxHP,
		CurrentHP:  maxHP,
		Initiative: initiative,
		Conditions: []string{},
		Dead:       false,
	})
	r.sortEntities()
	return nil
}

// ValidateAndRegister validates a join attempt and registers the client.
func (r *Room) ValidateAndRegister(role, dmToken, name string, conn *websocket.Conn) (*Client, int, string) {
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

func (r *Room) isNameTaken(name string) bool {
	for _, c := range r.Clients {
		if c.Role == "player" && c.Name == name {
			return true
		}
	}
	return false
}

// Registry is the global in-memory store of all active rooms.
type Registry struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

var Global = &Registry{rooms: make(map[string]*Room)}

// CreateRoom generates a unique room ID and DM token, registers the room, and returns both.
func (reg *Registry) CreateRoom() (roomID, dmToken string) {
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
		State:   RoomState{RoomID: roomID, Entities: []Entity{}},
		DMToken: dmToken,
		Clients: make(map[string]*Client),
	}
	return
}

// GetRoom retrieves a room by ID.
func (reg *Registry) GetRoom(roomID string) (*Room, bool) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	rm, ok := reg.rooms[roomID]
	return rm, ok
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
