package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/routes"
)

// TestPinsACT4_CreateActorHonoursCallerSuppliedServerOwnedFields pins a
// known defect (board row ACT4): CreateActor never clears a caller-supplied
// id, created_at or deleted before writing the row — actor_impl.go's
// CreateActor only defaults CreatedAt "if act.CreatedAt == \"\"" and never
// touches ID/Deleted at all before calling gormstore.ActorToRow(act).
// Driven through the real aggregator (routes.ActorRoutes -> a real
// http.ServeMux -> httptest.NewServer -> a real http.Client), not a direct
// manager call, so this also proves the gap is reachable over HTTP, not
// just through the Go API.
//
// This test must go RED once ACT4's fix lands (CreateActor must clear a
// caller-supplied ID/CreatedAt/Deleted before writing, the same way AG21
// fixed the identical shape in mwanachama-backend-agency) — flip the
// assertions below to check the response no longer echoes the caller's
// forged values.
func TestPinsACT4_CreateActorHonoursCallerSuppliedServerOwnedFields(t *testing.T) {
	um := newTestManager(t)
	mux := http.NewServeMux()
	for _, rt := range routes.ActorRoutes(um, routes.ResourceNames{}) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := srv.Client()

	const forgedID = "attacker-chosen-id"
	const forgedCreatedAt = "1999-01-01T00:00:00Z"
	body := `{"id":"` + forgedID + `","created_at":"` + forgedCreatedAt + `","deleted":true,"display_name":"Ghost"}`

	resp, err := client.Post(srv.URL+"/actors", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("pin: expected 201, got %d — ACT4 may already be fixed if this now rejects forged fields", resp.StatusCode)
	}
	var created struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		Deleted   bool   `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	// KNOWN DEFECT (ACT4): the caller's forged id/created_at/deleted are
	// echoed back verbatim instead of being server-minted/cleared.
	if created.ID != forgedID {
		t.Fatalf("pin: current (buggy) behaviour honours the caller's id verbatim; got %q, want %q — ACT4 may already be fixed", created.ID, forgedID)
	}
	if created.CreatedAt != forgedCreatedAt {
		t.Fatalf("pin: current (buggy) behaviour honours the caller's created_at verbatim; got %q, want %q — ACT4 may already be fixed", created.CreatedAt, forgedCreatedAt)
	}
	if !created.Deleted {
		t.Fatalf("pin: current (buggy) behaviour honours the caller's deleted:true verbatim; got false — ACT4 may already be fixed")
	}

	// KNOWN DEFECT (ACT4, part 2): the ghost row is invisible to ListActors
	// (it correctly filters deleted=true)...
	listResp, err := client.Get(srv.URL + "/actors")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.ID == forgedID {
			t.Fatalf("pin: expected the ghost actor to be filtered out of ListActors, but it was present")
		}
	}

	// ...but GetActor(id) still returns it directly (200, deleted:true),
	// because GetActor has no "WHERE deleted = false" filter the way
	// ListActors does — the row exists, is fully readable by anyone who
	// knows its id, yet is invisible to the roster. Nothing in this
	// package's own code ever legitimately sets Deleted=true (no Delete
	// method exists), so today the ONLY way a Deleted actor row can ever
	// exist is a caller choosing to POST one.
	getResp, err := client.Get(srv.URL + "/actors/" + forgedID)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("pin: expected GetActor to still return the ghost row directly (200), got %d — ACT4 may already be fixed", getResp.StatusCode)
	}
	var got struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Deleted {
		t.Fatalf("pin: expected GetActor to still report deleted:true for the ghost row")
	}
}

// TestPinsACT4_CreateGroupHonoursCallerSuppliedID pins the same defect
// shape in CreateGroup — group_impl.go's CreateGroup has the identical
// "if g.CreatedAt == \"\"" idiom and never clears g.ID either, so the same
// fix (or its own copy) is needed here too.
func TestPinsACT4_CreateGroupHonoursCallerSuppliedID(t *testing.T) {
	um := newTestManager(t)
	hc := stubHierarchy{hierarchies: map[string]bool{"h1": true}, levels: map[string]string{"l1": "h1"}}
	mux := http.NewServeMux()
	for _, rt := range routes.GroupRoutes(um, hc, routes.ResourceNames{}) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := srv.Client()

	const forgedID = "attacker-group-id"
	body := `{"id":"` + forgedID + `","hierarchy_id":"h1","level_id":"l1","name":"Ghost Group"}`
	resp, err := client.Post(srv.URL+"/groups", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("pin: expected 201, got %d — ACT4 may already be fixed for Group if this now rejects a forged id", resp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	// KNOWN DEFECT (ACT4): the caller's forged id is honoured verbatim
	// instead of a server-minted UUID.
	if created.ID != forgedID {
		t.Fatalf("pin: current (buggy) behaviour honours the caller's id verbatim; got %q, want %q — ACT4 may already be fixed for Group", created.ID, forgedID)
	}
}
