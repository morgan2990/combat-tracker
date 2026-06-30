package ws

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"combatapp/room"
	"combatapp/store"
	"github.com/gorilla/websocket"
)

const (
	pingInterval = 30 * time.Second
	readDeadline = 70 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --- Inbound message structs ---

type baseMsg struct {
	Type string `json:"type"`
}

// Player messages
type setupCharacterMsg struct{}

type setInitiativeMsg struct {
	Initiative int `json:"initiative"`
}

type updateEntityMsg struct {
	EntityID   string   `json:"entity_id"`
	CurrentHP  int      `json:"current_hp"`
	TempHP     int      `json:"temp_hp"`
	Conditions []string `json:"conditions"`
}

type addCompanionMsg struct {
	Name             string `json:"name"`
	MaxHP            int    `json:"max_hp"`
	SharesInitiative bool   `json:"shares_initiative"`
	Initiative       *int   `json:"initiative"`
}

// DM messages
type addCreatureMsg struct {
	Name               string `json:"name"`
	MaxHP              int    `json:"max_hp"`
	InitiativeModifier *int   `json:"initiative_modifier"`
	Quantity           int    `json:"quantity"`
	SourceType         string `json:"source_type"`
	ReferenceURL       string `json:"reference_url"`
	PDFObjectKey       string `json:"pdf_object_key"`
}

type removeEntityMsg struct {
	EntityID string `json:"entity_id"`
}

type dmUpdateEntityMsg struct {
	EntityID   string   `json:"entity_id"`
	Name       string   `json:"name"`
	CurrentHP  int      `json:"current_hp"`
	TempHP     int      `json:"temp_hp"`
	Initiative int      `json:"initiative"`
	Conditions []string `json:"conditions"`
	Dead       bool     `json:"dead"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	roomID := q.Get("room_id")
	name := q.Get("name")
	role := q.Get("role")
	dmToken := q.Get("dm_token")

	if roomID == "" || name == "" || (role != "dm" && role != "player") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	rm, found := room.Global.GetOrRestoreRoom(roomID, &store.GlobalRooms)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	if !found {
		closeWith(conn, 4004, "room not found")
		return
	}

	maxHP := 0
	if role == "player" {
		if raw := q.Get("max_hp"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil && v > 0 {
				maxHP = v
			}
		}
	}

	client, code, reason := rm.ValidateAndRegister(role, dmToken, name, conn, maxHP)
	if code != 0 {
		closeWith(conn, code, reason)
		return
	}

	rm.BroadcastState()
	go rm.PersistNow(&store.GlobalRooms)
	serve(rm, client)
}

func serve(rm *room.Room, c *room.Client) {
	defer func() {
		c.Conn.Close()
		rm.RemoveClient(c.SessionID)
		rm.BroadcastState()
		go rm.PersistNow(&store.GlobalRooms)
	}()

	c.Conn.SetReadDeadline(time.Now().Add(readDeadline))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	done := make(chan struct{})
	go pingLoop(c, done)
	defer close(done)

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		dispatch(rm, c, raw)
	}
}

func dispatch(rm *room.Room, c *room.Client, raw []byte) {
	var base baseMsg
	if err := json.Unmarshal(raw, &base); err != nil {
		return
	}

	switch base.Type {
	// --- Player actions ---
	case "setup_character":
		if c.MaxHP == 0 {
			return // no profile loaded
		}
		if err := rm.SetupCharacter(c.SessionID); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "set_initiative":
		var msg setInitiativeMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		if err := rm.SetInitiative(c.SessionID, msg.Initiative); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "update_entity":
		var msg updateEntityMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		if err := rm.UpdateEntity(c.SessionID, msg.EntityID, msg.CurrentHP, msg.TempHP, msg.Conditions); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "add_companion":
		var msg addCompanionMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		if err := rm.AddCompanion(c.SessionID, msg.Name, msg.MaxHP, msg.SharesInitiative, msg.Initiative); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "refresh_from_profile":
		if err := rm.RefreshFromProfile(c.SessionID, &store.Global); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	// --- DM actions ---
	case "start_combat":
		if err := rm.StartCombat(c.SessionID); err == nil {
			rm.BroadcastState()
			go rm.PersistNow(&store.GlobalRooms)
		}

	case "next_turn":
		if err := rm.NextTurn(c.SessionID); err == nil {
			rm.BroadcastState()
			go rm.PersistNow(&store.GlobalRooms)
		}

	case "add_creature":
		var msg addCreatureMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		if err := rm.AddCreature(c.SessionID, msg.Name, msg.MaxHP, msg.InitiativeModifier, msg.Quantity, msg.SourceType, msg.ReferenceURL, msg.PDFObjectKey); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "remove_entity":
		var msg removeEntityMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		if err := rm.RemoveEntity(c.SessionID, msg.EntityID); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "remove_dead_creatures":
		if err := rm.RemoveDeadCreatures(c.SessionID); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "dm_update_entity":
		var msg dmUpdateEntityMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		if err := rm.DMUpdateEntity(c.SessionID, msg.EntityID, msg.Name, msg.CurrentHP, msg.TempHP, msg.Initiative, msg.Conditions, msg.Dead); err == nil {
			rm.BroadcastState()
			rm.MarkDirty()
		}

	case "end_combat":
		if err := rm.EndCombat(c.SessionID); err == nil {
			rm.BroadcastState()
			go rm.PersistNow(&store.GlobalRooms)
		}
	}
}

func pingLoop(c *room.Client, done <-chan struct{}) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := c.WritePing(); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason))
	conn.Close()
}
