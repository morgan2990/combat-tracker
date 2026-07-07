//go:build integration

package store

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testTypesenseURL and testTypesenseAPIKey point at the real Typesense
// container started by TestMain. globalTypesense.client stays nil by
// default (matching production when Typesense is unconfigured/unreachable —
// see TestMonsterStore_SearchMonsters_NoTypesenseReturnsEmpty), so tests
// that need a real client build one from these and swap it in for their own
// duration (see withRealTypesense in typesense_integration_test.go).
const testTypesenseAPIKey = "test-api-key"

var testTypesenseURL string

// TestMain spins up a single, real MongoDB container plus a single, real
// Typesense container for the whole package's integration tests (not
// per-test — a fresh container per test would be needlessly slow), points
// store.Init() at Mongo via MONGODB_URI, and tears both down afterward. Run
// with: go test -tags=integration ./store/...
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
	if err := Init(); err != nil {
		panic("store.Init() against test container failed: " + err.Error())
	}

	tsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "typesense/typesense:27.1",
			ExposedPorts: []string{"8108/tcp"},
			Tmpfs:        map[string]string{"/data": ""},
			Cmd:          []string{"--data-dir=/data", "--api-key=" + testTypesenseAPIKey},
			WaitingFor:   wait.ForHTTP("/health").WithPort("8108/tcp"),
		},
		Started: true,
	})
	if err != nil {
		panic("failed to start Typesense container: " + err.Error())
	}
	defer func() {
		if err := tsContainer.Terminate(ctx); err != nil {
			panic("failed to terminate Typesense container: " + err.Error())
		}
	}()

	tsHost, err := tsContainer.Host(ctx)
	if err != nil {
		panic("failed to get Typesense container host: " + err.Error())
	}
	tsPort, err := tsContainer.MappedPort(ctx, "8108/tcp")
	if err != nil {
		panic("failed to get Typesense container port: " + err.Error())
	}
	testTypesenseURL = fmt.Sprintf("http://%s:%s", tsHost, tsPort.Port())

	os.Exit(m.Run())
}

func TestPartyStore_CreateAndGetByID(t *testing.T) {
	p, err := GlobalParties.CreateParty("The Fellowship")
	if err != nil {
		t.Fatalf("CreateParty() error = %v", err)
	}
	if p.ID == "" {
		t.Fatal("CreateParty() returned an empty ID")
	}
	if len(p.JoinCode) != joinCodeLength {
		t.Errorf("CreateParty() JoinCode = %q, want length %d", p.JoinCode, joinCodeLength)
	}
	if len(p.MemberPCIDs) != 0 {
		t.Errorf("CreateParty() MemberPCIDs = %v, want empty", p.MemberPCIDs)
	}

	got, err := GlobalParties.GetPartyByID(p.ID)
	if err != nil {
		t.Fatalf("GetPartyByID() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetPartyByID() = nil, want the party just created")
	}
	if got.Name != "The Fellowship" || got.JoinCode != p.JoinCode {
		t.Errorf("GetPartyByID() = %+v, want Name=The Fellowship JoinCode=%s", got, p.JoinCode)
	}
}

func TestPartyStore_GetPartyByID_NotFound(t *testing.T) {
	got, err := GlobalParties.GetPartyByID("does-not-exist")
	if err != nil {
		t.Fatalf("GetPartyByID() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("GetPartyByID() = %+v, want nil", got)
	}
}

func TestPartyStore_GetPartyByJoinCode(t *testing.T) {
	p, err := GlobalParties.CreateParty("Fellowship of the Ring")
	if err != nil {
		t.Fatalf("CreateParty() error = %v", err)
	}

	got, err := GlobalParties.GetPartyByJoinCode(p.JoinCode)
	if err != nil {
		t.Fatalf("GetPartyByJoinCode() error = %v", err)
	}
	if got == nil || got.ID != p.ID {
		t.Errorf("GetPartyByJoinCode(%q) = %+v, want party %s", p.JoinCode, got, p.ID)
	}

	if got, err := GlobalParties.GetPartyByJoinCode("ZZZZZ"); err != nil || got != nil {
		t.Errorf("GetPartyByJoinCode(unknown) = (%+v, %v), want (nil, nil)", got, err)
	}
}

func TestPartyStore_AddMember_IsIdempotent(t *testing.T) {
	p, err := GlobalParties.CreateParty("Grey Company")
	if err != nil {
		t.Fatalf("CreateParty() error = %v", err)
	}

	if _, err := GlobalParties.AddMember(p.ID, "pc1"); err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	got, err := GlobalParties.AddMember(p.ID, "pc1") // add the same member twice
	if err != nil {
		t.Fatalf("AddMember() (second call) error = %v", err)
	}
	if len(got.MemberPCIDs) != 1 {
		t.Errorf("MemberPCIDs = %v, want exactly one entry (AddMember should be idempotent)", got.MemberPCIDs)
	}
}

func TestPartyStore_AddMember_NotFound(t *testing.T) {
	got, err := GlobalParties.AddMember("does-not-exist", "pc1")
	if err != nil {
		t.Fatalf("AddMember() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("AddMember() on missing party = %+v, want nil", got)
	}
}

func TestPartyStore_UpdateParty(t *testing.T) {
	p, err := GlobalParties.CreateParty("Bree Company")
	if err != nil {
		t.Fatalf("CreateParty() error = %v", err)
	}

	currency := Currency{GP: 50, SP: 10}
	updated, err := GlobalParties.UpdateParty(p.ID, []string{"pc1", "pc2"}, currency)
	if err != nil {
		t.Fatalf("UpdateParty() error = %v", err)
	}
	if len(updated.MemberPCIDs) != 2 || updated.Currency != currency {
		t.Errorf("UpdateParty() = %+v, want MemberPCIDs=[pc1 pc2] Currency=%+v", updated, currency)
	}
}

func TestPartyStore_ListPartiesByMemberPCIDs(t *testing.T) {
	p1, err := GlobalParties.CreateParty("Party One")
	if err != nil {
		t.Fatalf("CreateParty() error = %v", err)
	}
	if _, err := GlobalParties.AddMember(p1.ID, "shared-pc"); err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}

	p2, err := GlobalParties.CreateParty("Party Two")
	if err != nil {
		t.Fatalf("CreateParty() error = %v", err)
	}
	if _, err := GlobalParties.AddMember(p2.ID, "other-pc"); err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}

	results, err := GlobalParties.ListPartiesByMemberPCIDs([]string{"shared-pc"})
	if err != nil {
		t.Fatalf("ListPartiesByMemberPCIDs() error = %v", err)
	}
	found := false
	for _, p := range results {
		if p.ID == p1.ID {
			found = true
		}
		if p.ID == p2.ID {
			t.Errorf("ListPartiesByMemberPCIDs([shared-pc]) unexpectedly included Party Two (%s)", p2.ID)
		}
	}
	if !found {
		t.Errorf("ListPartiesByMemberPCIDs([shared-pc]) = %+v, want it to include Party One (%s)", results, p1.ID)
	}

	empty, err := GlobalParties.ListPartiesByMemberPCIDs(nil)
	if err != nil {
		t.Fatalf("ListPartiesByMemberPCIDs(nil) error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("ListPartiesByMemberPCIDs(nil) = %v, want empty slice", empty)
	}
}
