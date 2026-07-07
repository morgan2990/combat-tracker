//go:build integration

package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"combatapp/room"
	"combatapp/store"
	"github.com/gorilla/websocket"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// TestMain spins up a single, real MongoDB container for the whole package's
// integration tests (not per-test — a fresh container per test would be
// needlessly slow), points store.Init() at it via MONGODB_URI, and tears it
// down afterward. Run with: go test -tags=integration ./ws/...
//
// This exists specifically to exercise the code paths in handler.go that
// touch store.Global/store.GlobalRooms/etc. directly (unseamed concrete
// types, unlike the api package's stores) — a plain unit test would panic
// the moment one of those methods is called against a nil collection. See
// handler_test.go for the paths that don't need a real store.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		panic("failed to start MongoDB container: " + err.Error())
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			panic("failed to terminate MongoDB container: " + err.Error())
		}
	}()

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		panic("failed to get MongoDB connection string: " + err.Error())
	}
	if err := os.Setenv("MONGODB_URI", uri); err != nil {
		panic("failed to set MONGODB_URI: " + err.Error())
	}
	if err := store.Init(); err != nil {
		panic("store.Init() against test container failed: " + err.Error())
	}

	os.Exit(m.Run())
}

// --- Handler() paths that require a real store ---

func TestHandler_PlayerForbidden_NotOwner_Closes4003(t *testing.T) {
	ownerUserID := "owner-" + store.NewID(6)
	otherUserID := "other-" + store.NewID(6)
	pc, err := store.Global.CreatePC(ownerUserID, "NotYours", 20)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}

	withFakeSessions(t, fakeSessionResolver{sessions: map[string]*store.Session{
		"tok-other": {Token: "tok-other", UserID: otherUserID, ExpiresAt: time.Now().Add(time.Hour)},
	}})

	srv := httptest.NewServer(http.HandlerFunc(Handler))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?room_id=someroom&role=player&pc_id=" + pc.ID
	header := http.Header{"Cookie": {"combatapp_session=tok-other"}}
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
	if closeErr.Code != 4003 {
		t.Errorf("close code = %d, want 4003", closeErr.Code)
	}
}

func TestHandler_RoomNotFound_Closes4004(t *testing.T) {
	dmUserID := "dm-" + store.NewID(6)
	withFakeSessions(t, fakeSessionResolver{sessions: map[string]*store.Session{
		"tok-dm": {Token: "tok-dm", UserID: dmUserID, ExpiresAt: time.Now().Add(time.Hour)},
	}})

	srv := httptest.NewServer(http.HandlerFunc(Handler))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?room_id=does-not-exist-" + store.NewID(6) + "&role=dm"
	header := http.Header{"Cookie": {"combatapp_session=tok-dm"}}
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
	if closeErr.Code != 4004 {
		t.Errorf("close code = %d, want 4004", closeErr.Code)
	}
}

// TestHandler_PlayerSuccess_FullLifecycle drives Handler()'s full success
// path — PC lookup, room lookup, ValidateAndRegister, the initial
// BroadcastState, and the async PersistNow — then serve()'s read loop (via a
// real setup_character message) and finally serve()'s deferred cleanup (via
// closing the connection), confirmed by polling for the persisted snapshot
// that cleanup's PersistNow call produces.
func TestHandler_PlayerSuccess_FullLifecycle(t *testing.T) {
	ownerUserID := "dm-owner-" + store.NewID(6)
	playerUserID := "player-" + store.NewID(6)

	pc, err := store.Global.CreatePC(playerUserID, "Aragorn", 30)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}

	roomID := room.Global.CreateRoom("5e", ownerUserID)
	t.Cleanup(func() { room.Global.RemoveRoom(roomID) })

	withFakeSessions(t, fakeSessionResolver{sessions: map[string]*store.Session{
		"tok-player": {Token: "tok-player", UserID: playerUserID, ExpiresAt: time.Now().Add(time.Hour)},
	}})

	srv := httptest.NewServer(http.HandlerFunc(Handler))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?room_id=" + roomID + "&role=player&pc_id=" + pc.ID
	header := http.Header{"Cookie": {"combatapp_session=tok-player"}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to dial test websocket server: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	var state map[string]any
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&state); err != nil {
		t.Fatalf("expected initial broadcast state, got error: %v", err)
	}
	if state["room_id"] != roomID {
		t.Errorf("broadcast room_id = %v, want %v", state["room_id"], roomID)
	}

	// Drive serve()'s read loop with a real message.
	if err := conn.WriteJSON(map[string]any{"type": "setup_character"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&state); err != nil {
		t.Fatalf("expected a broadcast after setup_character, got error: %v", err)
	}
	entities, _ := state["entities"].([]any)
	if len(entities) != 1 {
		t.Fatalf("got %d entities after setup_character, want 1", len(entities))
	}

	// Closing the client connection should trigger serve()'s deferred
	// cleanup, including an async PersistNow — poll for it rather than
	// asserting on unexported in-memory Room state. Handler()'s own connect
	// -time PersistNow (0 entities, fired before setup_character was even
	// sent) races the same document, so poll until the entity count matches
	// rather than stopping at the first snapshot found.
	conn.Close()
	var snap *store.RoomSnapshot
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snap, err = store.GlobalRooms.GetRoomSnapshot(roomID)
		if err == nil && snap != nil && len(snap.Entities) == 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if snap == nil {
		t.Fatal("expected a persisted room snapshot after serve() cleanup, got none")
	}
	if len(snap.Entities) != 1 {
		t.Errorf("persisted snapshot has %d entities, want 1", len(snap.Entities))
	}
}

// --- dispatch() cases that touch store.Global/store.GlobalRooms/etc. ---
//
// start_combat/next_turn/end_combat's success branches fire an async
// PersistNow(&store.GlobalRooms) — safe now that TestMain has pointed the
// store at a real MongoDB container, but a guaranteed panic in a plain unit
// test (see handler_test.go).

func TestDispatch_SetupCharacter(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	ownerUserID := "player-" + store.NewID(6)
	pc, err := store.Global.CreatePC(ownerUserID, "Aragorn", 30)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	if _, err := store.Global.CreateCompanion(pc.ID, ownerUserID, "Wolf", 11, true); err != nil {
		t.Fatalf("CreateCompanion() error = %v", err)
	}
	c.Role = "player"
	c.UserID = ownerUserID
	c.PCID = pc.ID
	c.Name = pc.Name
	c.MaxHP = pc.MaxHP

	dispatch(rm, c, []byte(`{"type":"setup_character"}`))

	state := readJSONWithTimeout(t, clientConn, 2*time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 2 {
		t.Fatalf("got %d entities, want 2 (pc + companion)", len(entities))
	}
}

func TestDispatch_RefreshFromProfile(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	ownerUserID := "player-" + store.NewID(6)
	pc, err := store.Global.CreatePC(ownerUserID, "Aragorn", 45)
	if err != nil {
		t.Fatalf("CreatePC() error = %v", err)
	}
	c.Role = "player"
	c.PCID = pc.ID
	addEntity(rm, room.Entity{ID: "e1", Type: "pc", SessionID: c.SessionID, MaxHP: 30, CurrentHP: 30})

	dispatch(rm, c, []byte(`{"type":"refresh_from_profile"}`))

	state := readJSONWithTimeout(t, clientConn, 2*time.Second)
	entities := state["entities"].([]any)
	e := entities[0].(map[string]any)
	if e["max_hp"] != float64(45) {
		t.Errorf("entity max_hp = %v, want 45 after refresh_from_profile", e["max_hp"])
	}
}

func TestDispatch_StartCombat(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)

	dispatch(rm, c, []byte(`{"type":"start_combat"}`))

	state := readJSONWithTimeout(t, clientConn, 2*time.Second)
	if state["is_started"] != true {
		t.Errorf("is_started = %v, want true", state["is_started"])
	}
}

func TestDispatch_NextTurn(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	rm.State.IsStarted = true

	dispatch(rm, c, []byte(`{"type":"next_turn"}`))

	state := readJSONWithTimeout(t, clientConn, 2*time.Second)
	if state["is_started"] != true {
		t.Errorf("is_started = %v, want true (unchanged by next_turn)", state["is_started"])
	}
}

func TestDispatch_EndCombat(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	rm.State.IsStarted = true

	dispatch(rm, c, []byte(`{"type":"end_combat"}`))

	state := readJSONWithTimeout(t, clientConn, 2*time.Second)
	if state["is_started"] != false {
		t.Errorf("is_started = %v, want false after end_combat", state["is_started"])
	}
}

func TestDispatch_InjectEncounter(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	c.UserID = "dm-" + store.NewID(6)

	monster, _, err := store.GlobalMonsters.UpsertMonster(store.Monster{
		Name: "TestGoblin-" + store.NewID(6), Edition: "5e", MaxHP: 7,
	})
	if err != nil {
		t.Fatalf("UpsertMonster() error = %v", err)
	}
	enc, err := store.GlobalEncounters.CreateEncounter(store.Encounter{
		Name:     "Test Encounter",
		OwnerID:  c.UserID,
		Edition:  "5e",
		Monsters: []store.EncounterMonster{{Name: monster.Name, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("CreateEncounter() error = %v", err)
	}

	dispatch(rm, c, []byte(`{"type":"inject_encounter","encounter_id":"`+enc.ID+`"}`))

	state := readJSONWithTimeout(t, clientConn, 2*time.Second)
	entities := state["entities"].([]any)
	if len(entities) != 2 {
		t.Fatalf("got %d entities, want 2 (quantity from encounter)", len(entities))
	}
}

func TestDispatch_InjectEncounter_UnknownEncounter_NoOp(t *testing.T) {
	rm, c, clientConn := newTestRoomAndClient(t)
	c.UserID = "dm-" + store.NewID(6)

	dispatch(rm, c, []byte(`{"type":"inject_encounter","encounter_id":"does-not-exist"}`))

	clientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var v map[string]any
	if err := clientConn.ReadJSON(&v); err == nil {
		t.Error("dispatch() with an unknown encounter_id should not broadcast anything")
	}
}
