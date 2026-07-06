package room

import (
	"testing"

	"combatapp/store"
)

func newTestRoom() *Room {
	return &Room{
		State:       RoomState{RoomID: "room1", Edition: "5e", Entities: []Entity{}},
		OwnerUserID: "dm1",
		Clients:     map[string]*Client{"dm-sess": {Role: "dm", SessionID: "dm-sess"}},
	}
}

// --- sortEntities ---

func TestSortEntities_DescendingByInitiative(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "e1", Initiative: new(10)},
		{ID: "e2", Initiative: new(20)},
		{ID: "e3", Initiative: new(15)},
	}
	r.sortEntities()

	got := []string{r.State.Entities[0].ID, r.State.Entities[1].ID, r.State.Entities[2].ID}
	want := []string{"e2", "e3", "e1"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("order = %v, want %v", got, want)
			break
		}
	}
}

func TestSortEntities_NilInitiativeSortsLast(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "e1", Initiative: nil},
		{ID: "e2", Initiative: new(5)},
	}
	r.sortEntities()

	if r.State.Entities[0].ID != "e2" || r.State.Entities[1].ID != "e1" {
		t.Errorf("nil-initiative entity should sort last, got order %v", r.State.Entities)
	}
}

func TestSortEntities_LairActionLosesTiebreak(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "lair", Type: "lair_action", Initiative: new(20)},
		{ID: "pc", Type: "pc", Initiative: new(20)},
	}
	r.sortEntities()

	if r.State.Entities[0].ID != "pc" {
		t.Errorf("expected lair_action to lose the tiebreak and sort after pc, got %v", r.State.Entities)
	}
}

func TestSortEntities_PreservesActiveEntityIndexByID(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	r.State.Entities = []Entity{
		{ID: "e1", Initiative: new(5)},
		{ID: "e2", Initiative: new(20)},
	}
	r.State.ActiveIndex = 0 // "e1" is active, currently at index 0
	r.sortEntities()

	// After sorting, e2 (20) comes before e1 (5); ActiveIndex should follow e1 to index 1.
	if r.State.Entities[r.State.ActiveIndex].ID != "e1" {
		t.Errorf("ActiveIndex = %d (%s), want it to still point at e1", r.State.ActiveIndex, r.State.Entities[r.State.ActiveIndex].ID)
	}
}

// --- StartCombat / NextTurn ---

func TestStartCombat_RejectsNonOwner(t *testing.T) {
	r := newTestRoom()
	if err := r.StartCombat("not-dm"); err == nil {
		t.Error("StartCombat() with non-owner session, want error")
	}
}

func TestStartCombat_RejectsIfAlreadyStarted(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	if err := r.StartCombat("dm-sess"); err == nil {
		t.Error("StartCombat() when already started, want error")
	}
}

func TestStartCombat_RejectsMissingPlayerInitiative(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "pc1", Type: "pc", Initiative: nil}}
	if err := r.StartCombat("dm-sess"); err == nil {
		t.Error("StartCombat() with a pc missing initiative, want error")
	}
}

func TestStartCombat_AutoRollsCreatureInitiative(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "c1", Type: "creature", InitiativeModifier: new(3)}}
	if err := r.StartCombat("dm-sess"); err != nil {
		t.Fatalf("StartCombat() error = %v", err)
	}
	e := r.State.Entities[0]
	if e.Initiative == nil || e.InitiativeRoll == nil {
		t.Fatal("creature should have Initiative and InitiativeRoll set after StartCombat")
	}
	if *e.Initiative != *e.InitiativeRoll+3 {
		t.Errorf("Initiative = %d, want InitiativeRoll(%d)+3", *e.Initiative, *e.InitiativeRoll)
	}
	if !r.State.IsStarted || r.State.Round != 1 || r.State.ActiveIndex != 0 {
		t.Errorf("state after start = %+v, want IsStarted=true Round=1 ActiveIndex=0", r.State)
	}
}

func TestNextTurn_RejectsNonOwner(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	if err := r.NextTurn("not-dm"); err == nil {
		t.Error("NextTurn() with non-owner, want error")
	}
}

func TestNextTurn_RejectsIfNotStarted(t *testing.T) {
	r := newTestRoom()
	if err := r.NextTurn("dm-sess"); err == nil {
		t.Error("NextTurn() before combat started, want error")
	}
}

func TestNextTurn_AdvancesIndex(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	r.State.Entities = []Entity{{ID: "e1"}, {ID: "e2"}}
	r.State.ActiveIndex = 0
	if err := r.NextTurn("dm-sess"); err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if r.State.ActiveIndex != 1 || r.State.Round != 0 {
		t.Errorf("ActiveIndex=%d Round=%d, want 1, 0", r.State.ActiveIndex, r.State.Round)
	}
}

func TestNextTurn_WrapsAndIncrementsRound(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	r.State.Round = 1
	r.State.Entities = []Entity{{ID: "e1"}, {ID: "e2"}}
	r.State.ActiveIndex = 1 // last entity
	if err := r.NextTurn("dm-sess"); err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if r.State.ActiveIndex != 0 || r.State.Round != 2 {
		t.Errorf("ActiveIndex=%d Round=%d, want 0, 2", r.State.ActiveIndex, r.State.Round)
	}
}

// --- AddCreature / appendCreatureGroup ---

func TestAddCreature_RejectsInvalidData(t *testing.T) {
	r := newTestRoom()
	if err := r.AddCreature("dm-sess", "", 10, nil, 1, "", "", "", ""); err == nil {
		t.Error("AddCreature() with empty name, want error")
	}
	if err := r.AddCreature("dm-sess", "Goblin", 0, nil, 1, "", "", "", ""); err == nil {
		t.Error("AddCreature() with maxHP=0, want error")
	}
}

func TestAddCreature_RejectsNonOwner(t *testing.T) {
	r := newTestRoom()
	if err := r.AddCreature("not-dm", "Goblin", 10, nil, 1, "", "", "", ""); err == nil {
		t.Error("AddCreature() with non-owner, want error")
	}
}

func TestAddCreature_QuantityNumbersNames(t *testing.T) {
	r := newTestRoom()
	if err := r.AddCreature("dm-sess", "Goblin", 7, nil, 3, "", "", "", ""); err != nil {
		t.Fatalf("AddCreature() error = %v", err)
	}
	if len(r.State.Entities) != 3 {
		t.Fatalf("got %d entities, want 3", len(r.State.Entities))
	}
	names := map[string]bool{}
	for _, e := range r.State.Entities {
		names[e.Name] = true
		if e.MaxHP != 7 || e.CurrentHP != 7 || e.Type != "creature" {
			t.Errorf("entity = %+v, want MaxHP=CurrentHP=7 Type=creature", e)
		}
	}
	for _, want := range []string{"Goblin 1", "Goblin 2", "Goblin 3"} {
		if !names[want] {
			t.Errorf("missing expected name %q among %v", want, names)
		}
	}
}

func TestAddCreature_SingleQuantityKeepsPlainName(t *testing.T) {
	r := newTestRoom()
	if err := r.AddCreature("dm-sess", "Goblin", 7, nil, 1, "", "", "", ""); err != nil {
		t.Fatalf("AddCreature() error = %v", err)
	}
	if r.State.Entities[0].Name != "Goblin" {
		t.Errorf("Name = %q, want %q", r.State.Entities[0].Name, "Goblin")
	}
}

func TestAddCreature_AutoRollsInitiativeWhenCombatStarted(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	if err := r.AddCreature("dm-sess", "Goblin", 7, new(2), 1, "", "", "", ""); err != nil {
		t.Fatalf("AddCreature() error = %v", err)
	}
	if r.State.Entities[0].Initiative == nil {
		t.Error("creature added mid-combat with an initiative modifier should get an initiative roll")
	}
}

// --- InjectEncounter ---

func TestInjectEncounter_RejectsNonOwner(t *testing.T) {
	r := newTestRoom()
	if err := r.InjectEncounter("not-dm", []MonsterGroup{{Name: "Orc", MaxHP: 15, Quantity: 1}}); err == nil {
		t.Error("InjectEncounter() with non-owner, want error")
	}
}

func TestInjectEncounter_AppendsAllGroups(t *testing.T) {
	r := newTestRoom()
	err := r.InjectEncounter("dm-sess", []MonsterGroup{
		{Name: "Orc", MaxHP: 15, Quantity: 2},
		{Name: "Goblin", MaxHP: 7, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("InjectEncounter() error = %v", err)
	}
	if len(r.State.Entities) != 3 {
		t.Errorf("got %d entities, want 3", len(r.State.Entities))
	}
}

// --- RemoveEntity ---

func TestRemoveEntity_NotFound(t *testing.T) {
	r := newTestRoom()
	if err := r.RemoveEntity("dm-sess", "missing"); err == nil {
		t.Error("RemoveEntity() for missing id, want error")
	}
}

func TestRemoveEntity_AdjustsActiveIndex_WhenRemovingBeforeActive(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1"}, {ID: "e2"}, {ID: "e3"}}
	r.State.ActiveIndex = 2
	if err := r.RemoveEntity("dm-sess", "e1"); err != nil {
		t.Fatalf("RemoveEntity() error = %v", err)
	}
	if r.State.ActiveIndex != 1 {
		t.Errorf("ActiveIndex = %d, want 1 (decremented)", r.State.ActiveIndex)
	}
}

func TestRemoveEntity_ResetsActiveIndex_WhenRemovingActiveLastEntity(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1"}, {ID: "e2"}}
	r.State.ActiveIndex = 1
	if err := r.RemoveEntity("dm-sess", "e2"); err != nil {
		t.Fatalf("RemoveEntity() error = %v", err)
	}
	if r.State.ActiveIndex != 0 {
		t.Errorf("ActiveIndex = %d, want 0 (reset, was out of bounds)", r.State.ActiveIndex)
	}
}

// --- ToggleEntityVisibility ---

func TestToggleEntityVisibility_NotFound(t *testing.T) {
	r := newTestRoom()
	if err := r.ToggleEntityVisibility("dm-sess", "missing"); err == nil {
		t.Error("ToggleEntityVisibility() for missing id, want error")
	}
}

func TestToggleEntityVisibility_Flips(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1", IsHidden: false}}
	if err := r.ToggleEntityVisibility("dm-sess", "e1"); err != nil {
		t.Fatalf("ToggleEntityVisibility() error = %v", err)
	}
	if !r.State.Entities[0].IsHidden {
		t.Error("IsHidden should be true after first toggle")
	}
	if err := r.ToggleEntityVisibility("dm-sess", "e1"); err != nil {
		t.Fatalf("ToggleEntityVisibility() error = %v", err)
	}
	if r.State.Entities[0].IsHidden {
		t.Error("IsHidden should be false after second toggle")
	}
}

// --- RemoveDeadCreatures ---

func TestRemoveDeadCreatures_NothingToRemove(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1", Type: "creature", Dead: false}}
	if err := r.RemoveDeadCreatures("dm-sess"); err == nil {
		t.Error("RemoveDeadCreatures() with nothing dead, want error")
	}
}

func TestRemoveDeadCreatures_RemovesOnlyDeadCreatures(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "e1", Type: "creature", Dead: true},
		{ID: "e2", Type: "creature", Dead: false},
		{ID: "e3", Type: "pc", Dead: true}, // dead PCs are never removed by this
	}
	if err := r.RemoveDeadCreatures("dm-sess"); err != nil {
		t.Fatalf("RemoveDeadCreatures() error = %v", err)
	}
	if len(r.State.Entities) != 2 {
		t.Fatalf("got %d entities, want 2", len(r.State.Entities))
	}
	for _, e := range r.State.Entities {
		if e.ID == "e1" {
			t.Error("e1 (dead creature) should have been removed")
		}
	}
}

// --- AddLairAction ---

func TestAddLairAction_DefaultsHiddenWithInitiative20(t *testing.T) {
	r := newTestRoom()
	if err := r.AddLairAction("dm-sess"); err != nil {
		t.Fatalf("AddLairAction() error = %v", err)
	}
	e := r.State.Entities[0]
	if e.Type != "lair_action" || !e.IsHidden || e.Initiative == nil || *e.Initiative != 20 {
		t.Errorf("lair action = %+v, want Type=lair_action IsHidden=true Initiative=20", e)
	}
}

// --- EndCombat ---

func TestEndCombat_RejectsIfNotStarted(t *testing.T) {
	r := newTestRoom()
	if err := r.EndCombat("dm-sess"); err == nil {
		t.Error("EndCombat() when not started, want error")
	}
}

func TestEndCombat_KeepsPCsAndOwnedCompanions_DropsCreatures(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	r.State.Entities = []Entity{
		{ID: "pc1", Type: "pc"},
		{ID: "comp1", Type: "companion", OwnerID: "pc1"},
		{ID: "comp-orphan", Type: "companion", OwnerID: "gone"},
		{ID: "creature1", Type: "creature"},
	}
	if err := r.EndCombat("dm-sess"); err != nil {
		t.Fatalf("EndCombat() error = %v", err)
	}
	ids := map[string]bool{}
	for _, e := range r.State.Entities {
		ids[e.ID] = true
	}
	if !ids["pc1"] || !ids["comp1"] {
		t.Errorf("expected pc1 and comp1 to survive, got %v", ids)
	}
	if ids["comp-orphan"] || ids["creature1"] {
		t.Errorf("expected comp-orphan and creature1 to be dropped, got %v", ids)
	}
	if r.State.IsStarted || r.State.Round != 0 || r.State.ActiveIndex != 0 {
		t.Errorf("state after EndCombat = %+v, want IsStarted=false Round=0 ActiveIndex=0", r.State)
	}
}

// --- DMUpdateEntity ---

func TestDMUpdateEntity_NotFound(t *testing.T) {
	r := newTestRoom()
	err := r.DMUpdateEntity("dm-sess", "missing", "", 0, 0, 0, nil, false, "")
	if err == nil {
		t.Error("DMUpdateEntity() for missing id, want error")
	}
}

func TestDMUpdateEntity_ClampsHP(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1", Type: "creature", MaxHP: 20}}
	if err := r.DMUpdateEntity("dm-sess", "e1", "", 999, -5, 0, nil, false, ""); err != nil {
		t.Fatalf("DMUpdateEntity() error = %v", err)
	}
	e := r.State.Entities[0]
	if e.CurrentHP != 20 || e.TempHP != 0 {
		t.Errorf("CurrentHP=%d TempHP=%d, want 20, 0 (clamped)", e.CurrentHP, e.TempHP)
	}
}

func TestDMUpdateEntity_NameOnlyAppliesToCreatures(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "pc1", Type: "pc", Name: "Original"}}
	if err := r.DMUpdateEntity("dm-sess", "pc1", "NewName", 0, 0, 0, nil, false, ""); err != nil {
		t.Fatalf("DMUpdateEntity() error = %v", err)
	}
	if r.State.Entities[0].Name != "Original" {
		t.Errorf("Name = %q, want unchanged %q (only creatures get renamed)", r.State.Entities[0].Name, "Original")
	}
}

func TestDMUpdateEntity_ResortsOnInitiativeChange(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "e1", Type: "creature", Initiative: new(5)},
		{ID: "e2", Type: "creature", Initiative: new(10)},
	}
	if err := r.DMUpdateEntity("dm-sess", "e1", "", 0, 0, 100, nil, false, ""); err != nil {
		t.Fatalf("DMUpdateEntity() error = %v", err)
	}
	if r.State.Entities[0].ID != "e1" {
		t.Errorf("expected e1 (now initiative 100) to sort first, got %v", r.State.Entities)
	}
}

// --- SetupCharacter ---

func TestSetupCharacter_RejectsNonPlayerSession(t *testing.T) {
	r := newTestRoom()
	if err := r.SetupCharacter("dm-sess"); err == nil {
		t.Error("SetupCharacter() for a DM session, want error")
	}
}

func TestSetupCharacter_RejectsNoPCLoaded(t *testing.T) {
	r := newTestRoom()
	r.Clients["p1"] = &Client{Role: "player", SessionID: "p1", MaxHP: 0}
	if err := r.SetupCharacter("p1"); err == nil {
		t.Error("SetupCharacter() with MaxHP=0, want error")
	}
}

func TestSetupCharacter_RejectsDuplicateSetup(t *testing.T) {
	r := newTestRoom()
	r.Clients["p1"] = &Client{Role: "player", SessionID: "p1", MaxHP: 20}
	r.State.Entities = []Entity{{ID: "existing", SessionID: "p1"}}
	if err := r.SetupCharacter("p1"); err == nil {
		t.Error("SetupCharacter() when already set up, want error")
	}
}

func TestSetupCharacter_Success(t *testing.T) {
	r := newTestRoom()
	r.Clients["p1"] = &Client{Role: "player", SessionID: "p1", Name: "Frodo", PCID: "pc1", MaxHP: 20}
	if err := r.SetupCharacter("p1"); err != nil {
		t.Fatalf("SetupCharacter() error = %v", err)
	}
	if len(r.State.Entities) != 1 {
		t.Fatalf("got %d entities, want 1", len(r.State.Entities))
	}
	e := r.State.Entities[0]
	if e.Type != "pc" || e.Name != "Frodo" || e.MaxHP != 20 || e.CurrentHP != 20 || e.PCID != "pc1" {
		t.Errorf("created entity = %+v, want Type=pc Name=Frodo MaxHP=CurrentHP=20 PCID=pc1", e)
	}
}

// --- InstantiateCompanionsFromPC ---

func TestInstantiateCompanionsFromPC_RequiresExistingPCEntity(t *testing.T) {
	r := newTestRoom()
	err := r.InstantiateCompanionsFromPC("p1", []store.PC{{ID: "c1", Name: "Brego"}})
	if err == nil {
		t.Error("InstantiateCompanionsFromPC() with no PC entity for the session, want error")
	}
}

func TestInstantiateCompanionsFromPC_AppendsEachCompanion(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "pc-entity", Type: "pc", SessionID: "p1"}}
	companions := []store.PC{
		{ID: "c1", Name: "Brego", MaxHP: 15, SharesInitiative: true},
		{ID: "c2", Name: "Bill", MaxHP: 10},
	}
	if err := r.InstantiateCompanionsFromPC("p1", companions); err != nil {
		t.Fatalf("InstantiateCompanionsFromPC() error = %v", err)
	}
	if len(r.State.Entities) != 3 {
		t.Fatalf("got %d entities, want 3 (1 pc + 2 companions)", len(r.State.Entities))
	}
}

// --- SetInitiative ---

func TestSetInitiative_RequiresExistingPCEntity(t *testing.T) {
	r := newTestRoom()
	if err := r.SetInitiative("p1", 15); err == nil {
		t.Error("SetInitiative() with no PC entity, want error")
	}
}

func TestSetInitiative_PropagatesToSharedCompanionsOnly(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "pc-entity", Type: "pc", SessionID: "p1"},
		{ID: "comp-shared", Type: "companion", OwnerID: "pc-entity", SharesInitiative: true},
		{ID: "comp-own", Type: "companion", OwnerID: "pc-entity", SharesInitiative: false},
	}
	if err := r.SetInitiative("p1", 15); err != nil {
		t.Fatalf("SetInitiative() error = %v", err)
	}
	byID := map[string]Entity{}
	for _, e := range r.State.Entities {
		byID[e.ID] = e
	}
	if byID["pc-entity"].Initiative == nil || *byID["pc-entity"].Initiative != 15 {
		t.Error("PC entity's initiative should be set to 15")
	}
	if byID["comp-shared"].Initiative == nil || *byID["comp-shared"].Initiative != 15 {
		t.Error("shared-initiative companion should inherit 15")
	}
	if byID["comp-own"].Initiative != nil {
		t.Error("non-shared companion's initiative should be untouched")
	}
}

// --- UpdateEntity ---

func TestUpdateEntity_RejectsUnownedEntity(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1", SessionID: "someone-else"}}
	err := r.UpdateEntity("p1", "e1", 10, 0, nil)
	if err == nil {
		t.Error("UpdateEntity() on an entity the session doesn't own, want error")
	}
}

func TestUpdateEntity_AllowsOwnCompanion(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{
		{ID: "pc-entity", Type: "pc", SessionID: "p1"},
		{ID: "comp1", Type: "companion", OwnerID: "pc-entity", MaxHP: 20},
	}
	if err := r.UpdateEntity("p1", "comp1", 15, 0, nil); err != nil {
		t.Fatalf("UpdateEntity() on own companion, error = %v", err)
	}
	if r.State.Entities[1].CurrentHP != 15 {
		t.Errorf("CurrentHP = %d, want 15", r.State.Entities[1].CurrentHP)
	}
}

func TestUpdateEntity_ClampsHP(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "e1", SessionID: "p1", MaxHP: 20}}
	if err := r.UpdateEntity("p1", "e1", 999, -10, nil); err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}
	if r.State.Entities[0].CurrentHP != 20 || r.State.Entities[0].TempHP != 0 {
		t.Errorf("CurrentHP=%d TempHP=%d, want 20, 0", r.State.Entities[0].CurrentHP, r.State.Entities[0].TempHP)
	}
}

// --- AddCompanion ---

func TestAddCompanion_RejectsInvalidData(t *testing.T) {
	r := newTestRoom()
	if err := r.AddCompanion("p1", "", 10, false, nil); err == nil {
		t.Error("AddCompanion() with empty name, want error")
	}
	if err := r.AddCompanion("p1", "Pet", 0, false, nil); err == nil {
		t.Error("AddCompanion() with maxHP=0, want error")
	}
}

func TestAddCompanion_RequiresExistingPCEntity(t *testing.T) {
	r := newTestRoom()
	if err := r.AddCompanion("p1", "Pet", 5, false, nil); err == nil {
		t.Error("AddCompanion() with no PC entity for the session, want error")
	}
}

func TestAddCompanion_Success(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "pc-entity", Type: "pc", SessionID: "p1"}}
	if err := r.AddCompanion("p1", "Pet", 5, true, new(8)); err != nil {
		t.Fatalf("AddCompanion() error = %v", err)
	}
	if len(r.State.Entities) != 2 {
		t.Fatalf("got %d entities, want 2", len(r.State.Entities))
	}
	var comp Entity
	for _, e := range r.State.Entities {
		if e.Type == "companion" {
			comp = e
		}
	}
	if comp.OwnerID != "pc-entity" || comp.MaxHP != 5 || !comp.SharesInitiative {
		t.Errorf("companion = %+v, want OwnerID=pc-entity MaxHP=5 SharesInitiative=true", comp)
	}
}

// --- ValidateAndRegister ---

func TestValidateAndRegister_RejectsNonOwnerDM(t *testing.T) {
	r := newTestRoom()
	_, code, _ := r.ValidateAndRegister("dm", "not-the-owner", "", "", 0, nil)
	if code != 4003 {
		t.Errorf("code = %d, want 4003", code)
	}
}

func TestValidateAndRegister_RejectsAlreadyActivePC(t *testing.T) {
	r := newTestRoom()
	r.Clients["existing"] = &Client{Role: "player", PCID: "pc1"}
	_, code, _ := r.ValidateAndRegister("player", "user1", "pc1", "Frodo", 20, nil)
	if code != 4009 {
		t.Errorf("code = %d, want 4009", code)
	}
}

func TestValidateAndRegister_Success_LinksExistingEntity(t *testing.T) {
	r := newTestRoom()
	r.State.Entities = []Entity{{ID: "pc-entity", Type: "pc", PCID: "pc1"}}
	c, code, msg := r.ValidateAndRegister("player", "user1", "pc1", "Frodo", 20, nil)
	if code != 0 || msg != "" {
		t.Fatalf("code=%d msg=%q, want 0, \"\"", code, msg)
	}
	if c.SessionID == "" {
		t.Fatal("client should have a generated SessionID")
	}
	if r.State.Entities[0].SessionID != c.SessionID {
		t.Error("existing PC entity should be linked to the new session")
	}
	if _, ok := r.Clients[c.SessionID]; !ok {
		t.Error("client should be registered in r.Clients")
	}
}

func TestValidateAndRegister_DMOwnerSucceeds(t *testing.T) {
	r := newTestRoom()
	c, code, _ := r.ValidateAndRegister("dm", "dm1", "", "DM", 0, nil)
	if code != 0 || c.Role != "dm" {
		t.Errorf("code=%d role=%q, want 0, dm", code, c.Role)
	}
}

// --- RemoveClient / Summary / isPCActive / MarkDirty ---

func TestRemoveClient(t *testing.T) {
	r := newTestRoom()
	r.Clients["p1"] = &Client{SessionID: "p1"}
	r.RemoveClient("p1")
	if _, ok := r.Clients["p1"]; ok {
		t.Error("client should have been removed")
	}
}

func TestSummary(t *testing.T) {
	r := newTestRoom()
	r.State.IsStarted = true
	id, edition, active := r.Summary()
	if id != "room1" || edition != "5e" || !active {
		t.Errorf("Summary() = (%q, %q, %v), want (room1, 5e, true)", id, edition, active)
	}
}

func TestIsPCActive(t *testing.T) {
	r := newTestRoom()
	r.Clients["p1"] = &Client{Role: "player", PCID: "pc1"}
	if !r.isPCActive("pc1") {
		t.Error("isPCActive(pc1) = false, want true")
	}
	if r.isPCActive("pc2") {
		t.Error("isPCActive(pc2) = true, want false")
	}
}

func TestMarkDirty(t *testing.T) {
	r := newTestRoom()
	if r.dirty {
		t.Fatal("new room should not start dirty")
	}
	r.MarkDirty()
	if !r.dirty {
		t.Error("MarkDirty() should set dirty=true")
	}
}

// --- ClaimParty / PartyID ---

func TestClaimParty_FirstCallSucceeds(t *testing.T) {
	r := newTestRoom()
	if !r.ClaimParty("party1") {
		t.Error("first ClaimParty() call should succeed")
	}
	if r.PartyID() != "party1" {
		t.Errorf("PartyID() = %q, want party1", r.PartyID())
	}
}

func TestClaimParty_SecondCallFails(t *testing.T) {
	r := newTestRoom()
	r.ClaimParty("party1")
	if r.ClaimParty("party2") {
		t.Error("second ClaimParty() call should fail (already claimed)")
	}
	if r.PartyID() != "party1" {
		t.Errorf("PartyID() = %q, want it to remain party1", r.PartyID())
	}
}

func TestClaimPartyIfUnclaimed_NoOpWhenAlreadyClaimed(t *testing.T) {
	r := newTestRoom()
	r.ClaimParty("party1")
	// Passing a nil *store.PartyStore is safe here because the "already
	// claimed" branch returns before ever touching ps.
	if r.ClaimPartyIfUnclaimed("pc1", nil) {
		t.Error("ClaimPartyIfUnclaimed() should be a no-op when already claimed")
	}
}

// --- Registry ---

func TestRegistry_CreateAndGetRoom(t *testing.T) {
	reg := &Registry{rooms: make(map[string]*Room)}
	id := reg.CreateRoom("5e", "owner1")
	if len(id) != 5 {
		t.Errorf("CreateRoom() id = %q, want length 5", id)
	}
	rm, ok := reg.GetRoom(id)
	if !ok || rm.OwnerUserID != "owner1" || rm.State.Edition != "5e" {
		t.Errorf("GetRoom(%q) = (%+v, %v), want OwnerUserID=owner1 Edition=5e", id, rm, ok)
	}
}

func TestRegistry_GetRoom_NotFound(t *testing.T) {
	reg := &Registry{rooms: make(map[string]*Room)}
	_, ok := reg.GetRoom("does-not-exist")
	if ok {
		t.Error("GetRoom() for a missing room, want ok=false")
	}
}

func TestRegistry_RemoveRoom(t *testing.T) {
	reg := &Registry{rooms: make(map[string]*Room)}
	id := reg.CreateRoom("5e", "owner1")
	reg.RemoveRoom(id) // room has zero clients, so Close() has nothing to iterate
	if _, ok := reg.GetRoom(id); ok {
		t.Error("room should have been removed from the registry")
	}
}

func TestRegistry_RemoveRoom_NoOpWhenMissing(t *testing.T) {
	reg := &Registry{rooms: make(map[string]*Room)}
	reg.RemoveRoom("does-not-exist") // must not panic
}

// --- inflateRoom ---

func TestInflateRoom_RoundTripsEntitiesAndResolvesActiveIndex(t *testing.T) {
	snap := store.RoomSnapshot{
		RoomID:             "room1",
		OwnerUserID:        "owner1",
		Edition:            "5e",
		IsCombatActive:     true,
		CurrentRound:       2,
		ActiveTurnEntityID: new("e2"),
		Entities: []store.RoomEntitySnapshot{
			{ID: "e1", Name: "First"},
			{ID: "e2", Name: "Second"},
		},
	}
	rm := inflateRoom(snap)

	if rm.State.RoomID != "room1" || rm.OwnerUserID != "owner1" || !rm.State.IsStarted || rm.State.Round != 2 {
		t.Errorf("inflated room = %+v, unexpected top-level fields", rm.State)
	}
	if len(rm.State.Entities) != 2 {
		t.Fatalf("got %d entities, want 2", len(rm.State.Entities))
	}
	if rm.State.ActiveIndex != 1 {
		t.Errorf("ActiveIndex = %d, want 1 (resolved from e2)", rm.State.ActiveIndex)
	}
	if rm.Clients == nil || len(rm.Clients) != 0 {
		t.Error("inflated room should have an empty, non-nil Clients map")
	}
}

// --- newRoomID ---

func TestNewRoomID_FormatAndCharset(t *testing.T) {
	for range 50 {
		id := newRoomID()
		if len(id) != 5 {
			t.Fatalf("newRoomID() = %q, want length 5", id)
		}
		for _, c := range id {
			found := false
			for _, allowed := range idChars {
				if c == allowed {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("newRoomID() = %q contains disallowed character %q", id, c)
			}
		}
	}
}
