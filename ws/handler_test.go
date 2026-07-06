package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"combatapp/room"
	"github.com/gorilla/websocket"
)

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
