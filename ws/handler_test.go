package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"combatapp/auth"
	"combatapp/room"
	"combatapp/store"
	"github.com/gorilla/websocket"
)

// fakeSessionResolver is an in-memory auth.SessionResolver, mirroring the one
// in auth/auth_test.go — duplicated here since it's unexported there.
type fakeSessionResolver struct {
	sessions map[string]*store.Session
}

func (f fakeSessionResolver) ResolveSession(token string) (*store.Session, error) {
	return f.sessions[token], nil
}

// withFakeSessions swaps auth.Sessions for fake for the duration of the test.
func withFakeSessions(t *testing.T, fake auth.SessionResolver) {
	t.Helper()
	original := auth.Sessions
	auth.Sessions = fake
	t.Cleanup(func() { auth.Sessions = original })
}

// newTestWSConnPair starts a real (loopback, no Docker needed) WebSocket
// handshake and returns the server-side connection to attach to a
// room.Client, plus the client-side connection so tests can read whatever
// the server side writes (e.g. via BroadcastState).
func newTestWSConnPair(t *testing.T) (serverConn, clientConn *websocket.Conn) {
	t.Helper()
	var upgrader websocket.Upgrader
	connCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connCh <- conn
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial test websocket server: %v", err)
	}
	t.Cleanup(func() { clientConn.Close() })

	select {
	case serverConn = <-connCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server-side websocket upgrade")
	}
	t.Cleanup(func() { serverConn.Close() })
	return serverConn, clientConn
}

func newTestRoomAndClient(t *testing.T) (*room.Room, *room.Client, *websocket.Conn) {
	t.Helper()
	serverConn, clientConn := newTestWSConnPair(t)
	c := &room.Client{Conn: serverConn, Role: "dm", SessionID: "dm-sess"}
	rm := &room.Room{
		State:       room.RoomState{RoomID: "room1", Edition: "5e", Entities: []room.Entity{}},
		OwnerUserID: "owner1",
		Clients:     map[string]*room.Client{c.SessionID: c},
	}
	return rm, c, clientConn
}

// addEntity appends an entity directly to rm's state — dispatch's guard
// clauses expect entities to already exist in the room, same as they would
// after an earlier action (e.g. setup_character) in production.
func addEntity(rm *room.Room, e room.Entity) {
	rm.State.Entities = append(rm.State.Entities, e)
}

// readJSONWithTimeout reads one JSON message from conn, failing the test if
// none arrives within the timeout — used to confirm dispatch actually
// broadcast state after a successful action.
func readJSONWithTimeout(t *testing.T, conn *websocket.Conn, timeout time.Duration) map[string]any {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	var v map[string]any
	if err := conn.ReadJSON(&v); err != nil {
		t.Fatalf("expected a broadcast message, got error: %v", err)
	}
	return v
}

func TestDispatch_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	dispatch(rm, c, []byte("not json"))

	// Nothing should have been broadcast — confirm no message arrives promptly.
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed JSON should not broadcast anything")
	}
}

func TestDispatch_UnknownType_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	dispatch(rm, c, []byte(`{"type":"not_a_real_action"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with an unrecognized type should not broadcast anything")
	}
}

func TestDispatch_SetInitiative(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "pc1", Type: "pc", SessionID: c.SessionID})

	dispatch(rm, c, []byte(`{"type":"set_initiative","initiative":12}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	got := entities[0].(map[string]any)["initiative"]
	if got != float64(12) {
		t.Errorf("broadcast entity initiative = %v, want 12", got)
	}
}

func TestDispatch_UpdateEntity(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1", SessionID: c.SessionID, MaxHP: 20})

	dispatch(rm, c, []byte(`{"type":"update_entity","entity_id":"e1","current_hp":15,"temp_hp":2,"conditions":["Prone"]}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	e := entities[0].(map[string]any)
	if e["current_hp"] != float64(15) || e["temp_hp"] != float64(2) {
		t.Errorf("broadcast entity = %+v, want current_hp=15 temp_hp=2", e)
	}
}

func TestDispatch_AddCompanion(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "pc1", Type: "pc", SessionID: c.SessionID})

	dispatch(rm, c, []byte(`{"type":"add_companion","name":"Wolf","max_hp":11,"shares_initiative":true}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 2 {
		t.Fatalf("got %d entities, want 2", len(entities))
	}
}

func TestDispatch_RemoveEntity(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1"})

	dispatch(rm, c, []byte(`{"type":"remove_entity","entity_id":"e1"}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 0 {
		t.Errorf("got %d entities, want 0 after removal", len(entities))
	}
}

func TestDispatch_RemoveDeadCreatures(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "c1", Type: "creature", Dead: true})

	dispatch(rm, c, []byte(`{"type":"remove_dead_creatures"}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 0 {
		t.Errorf("got %d entities, want 0 after removing dead creatures", len(entities))
	}
}

func TestDispatch_RemoveDeadCreatures_NothingDead_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "c1", Type: "creature", Dead: false})

	dispatch(rm, c, []byte(`{"type":"remove_dead_creatures"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() should not broadcast when RemoveDeadCreatures errors (nothing to remove)")
	}
}

func TestDispatch_AddLairAction(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)

	dispatch(rm, c, []byte(`{"type":"add_lair_action"}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 1 || entities[0].(map[string]any)["type"] != "lair_action" {
		t.Errorf("broadcast entities = %v, want one lair_action", entities)
	}
}

func TestDispatch_ToggleEntityVisibility(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1", IsHidden: false})

	dispatch(rm, c, []byte(`{"type":"toggle_entity_visibility","entity_id":"e1"}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	if entities[0].(map[string]any)["is_hidden"] != true {
		t.Error("expected is_hidden to be true after toggle")
	}
}

func TestDispatch_DMUpdateEntity(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1", Type: "creature", MaxHP: 30})

	dispatch(rm, c, []byte(`{"type":"dm_update_entity","entity_id":"e1","name":"Renamed","current_hp":10,"temp_hp":0,"initiative":5,"conditions":[],"dead":false}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	e := entities[0].(map[string]any)
	if e["name"] != "Renamed" || e["current_hp"] != float64(10) {
		t.Errorf("broadcast entity = %+v, want name=Renamed current_hp=10", e)
	}
}

func TestCloseWith(t *testing.T) {
	serverConn, clientConn := newTestWSConnPair(t)
	closeWith(serverConn, 4001, "test reason")

	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err := clientConn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("expected a CloseError, got %v (%T)", err, err)
	}
	if closeErr.Code != 4001 || closeErr.Text != "test reason" {
		t.Errorf("close = (%d, %q), want (4001, \"test reason\")", closeErr.Code, closeErr.Text)
	}
}

func TestBaseMsgUnmarshal(t *testing.T) {
	var base baseMsg
	if err := json.Unmarshal([]byte(`{"type":"set_initiative","initiative":5}`), &base); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if base.Type != "set_initiative" {
		t.Errorf("Type = %q, want set_initiative", base.Type)
	}
}

// --- Handler() request-validation and auth-failure branches ---
//
// These are the only Handler() branches reachable without a live MongoDB
// connection: everything past a successful auth check touches either
// store.Global (PC lookup, for role=player) or room.Global.GetOrRestoreRoom's
// MongoDB fallback (for an unregistered room), both of which are concrete,
// unseamed store types. Those paths are covered by the integration tests in
// handler_integration_test.go instead.

func TestHandler_MissingRoomID_BadRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws?role=dm", nil)
	w := httptest.NewRecorder()
	Handler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_InvalidRole_BadRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws?room_id=room1&role=spectator", nil)
	w := httptest.NewRecorder()
	Handler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_PlayerMissingPCID_BadRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws?room_id=room1&role=player", nil)
	w := httptest.NewRecorder()
	Handler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_AuthFailure_Closes4001(t *testing.T) {
	withFakeSessions(t, fakeSessionResolver{sessions: map[string]*store.Session{}})

	srv := httptest.NewServer(http.HandlerFunc(Handler))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?room_id=room1&role=dm"
	header := http.Header{"Cookie": {"combatapp_session=tok-unknown"}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to dial test websocket server: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("expected a CloseError, got %v (%T)", err, err)
	}
	if closeErr.Code != 4001 {
		t.Errorf("close code = %d, want 4001", closeErr.Code)
	}
}

// --- dispatch() malformed-JSON branches ---
//
// Each already-tested message type also has an early-return branch when the
// payload doesn't unmarshal into its expected struct. A non-object JSON
// value (a bare number) fails json.Unmarshal into any of these structs.

func TestDispatch_SetInitiative_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "pc1", SessionID: c.SessionID})

	dispatch(rm, c, []byte(`{"type":"set_initiative","initiative":"nope"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed set_initiative payload should not broadcast anything")
	}
}

func TestDispatch_UpdateEntity_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1", SessionID: c.SessionID})

	dispatch(rm, c, []byte(`{"type":"update_entity","current_hp":"nope"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed update_entity payload should not broadcast anything")
	}
}

func TestDispatch_AddCompanion_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)

	dispatch(rm, c, []byte(`{"type":"add_companion","max_hp":"nope"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed add_companion payload should not broadcast anything")
	}
}

func TestDispatch_RemoveEntity_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1"})

	dispatch(rm, c, []byte(`{"type":"remove_entity","entity_id":123}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed remove_entity payload should not broadcast anything")
	}
}

func TestDispatch_ToggleEntityVisibility_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1"})

	dispatch(rm, c, []byte(`{"type":"toggle_entity_visibility","entity_id":123}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed toggle_entity_visibility payload should not broadcast anything")
	}
}

func TestDispatch_DMUpdateEntity_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	addEntity(rm, room.Entity{ID: "e1"})

	dispatch(rm, c, []byte(`{"type":"dm_update_entity","current_hp":"nope"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed dm_update_entity payload should not broadcast anything")
	}
}

// --- dispatch() "add_creature" ---
//
// Unlike start_combat/next_turn/add_creature's DM siblings, a successful
// add_creature only broadcasts + marks dirty — it does not fire the async
// PersistNow(&store.GlobalRooms) call, so it's safe to exercise here without
// a live MongoDB connection.

func TestDispatch_AddCreature_MalformedJSON_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)

	dispatch(rm, c, []byte(`{"type":"add_creature","max_hp":"nope"}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with malformed add_creature payload should not broadcast anything")
	}
}

func TestDispatch_AddCreature(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)

	dispatch(rm, c, []byte(`{"type":"add_creature","name":"Goblin","max_hp":7,"quantity":2}`))

	state := readJSONWithTimeout(t, clientConn, time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 2 {
		t.Fatalf("got %d entities, want 2", len(entities))
	}
}

func TestDispatch_AddCreature_InvalidData_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)

	// AddCreature() rejects an empty name before ever touching room state.
	dispatch(rm, c, []byte(`{"type":"add_creature","name":"","max_hp":7}`))

	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() should not broadcast when AddCreature errors (invalid data)")
	}
}

// --- pingLoop() ---

func TestPingLoop_ReturnsWhenDoneClosed(t *testing.T) {
	serverConn, clientConn := newTestWSConnPair(t)
	c := &room.Client{Conn: serverConn, Role: "dm", SessionID: "dm-sess"}
	_ = clientConn

	done := make(chan struct{})
	loopReturned := make(chan struct{})
	go func() {
		pingLoop(c, done)
		close(loopReturned)
	}()

	close(done)
	select {
	case <-loopReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("pingLoop() did not return promptly after done was closed")
	}
}
