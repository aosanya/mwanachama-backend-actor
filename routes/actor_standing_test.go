package routes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

type standingFixture struct {
	mgr                 mwanachamaactor.UserManager
	srv                 *httptest.Server
	asha, grace, peter  mwanachamaactor.Actor
	wardA, wardB        mwanachamaactor.Group
	farmer, coordinator mwanachamaactor.RoleKind
}

func newStandingFixture(t *testing.T) standingFixture {
	t.Helper()
	mgr := newManager(t)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	var f standingFixture
	f.mgr = mgr
	var err error
	f.asha, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Asha Otieno"})
	must(err)
	f.grace, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Grace Adhiambo"})
	must(err)
	f.peter, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Peter Mwangi"})
	must(err)
	f.wardA, err = mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward A"})
	must(err)
	f.wardB, err = mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward B"})
	must(err)
	f.farmer, err = mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Farmer", DefinitionRoleID: "def-farmer"})
	must(err)
	f.coordinator, err = mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator", DefinitionRoleID: "def-coordinator", LeadsGroup: true, Staff: true})
	must(err)
	for _, r := range []struct{ actor, group string }{
		{f.asha.ID, f.wardA.ID},
		{f.grace.ID, f.wardA.ID},
		{f.grace.ID, f.wardB.ID},
	} {
		_, err = mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: r.actor, GroupID: r.group})
		must(err)
	}
	f.srv = newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return f.peter.ID },
	}))
	return f
}

func (f standingFixture) grantSeat(t *testing.T, actorID, groupID, kindID string) string {
	t.Helper()
	status, seat := do(t, f.srv, http.MethodPost, "/role-assignments", map[string]any{
		"actor_id": actorID, "group_id": groupID, "kind_id": kindID,
	})
	id, _ := seat["id"].(string)
	if status != http.StatusCreated || id == "" {
		t.Fatalf("POST /role-assignments = %d %v, want 201", status, seat)
	}
	return id
}

func standingOf(t *testing.T, srv *httptest.Server, actorID string) (int, models.ActorStanding) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/actors/" + actorID + "/standing")
	if err != nil {
		t.Fatalf("GET standing: %v", err)
	}
	defer resp.Body.Close()
	var out models.ActorStanding
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestTwoPeopleAnswerDifferentlyOnTheirStanding(t *testing.T) {
	f := newStandingFixture(t)
	f.grantSeat(t, f.asha.ID, f.wardA.ID, f.farmer.ID)
	f.grantSeat(t, f.grace.ID, f.wardA.ID, f.coordinator.ID)

	status, asha := standingOf(t, f.srv, f.asha.ID)
	if status != http.StatusOK {
		t.Fatalf("GET Asha's standing = %d, want 200", status)
	}
	if asha.DisplayName != "Asha Otieno" || len(asha.Registrations) != 1 || asha.Registrations[0].GroupName != "Ward A" || asha.Registrations[0].Since == "" {
		t.Fatalf("Asha's registrations = %+v, want only Ward A with a since", asha)
	}
	if len(asha.Seats) != 1 {
		t.Fatalf("Asha's seats = %+v, want only Farmer", asha.Seats)
	}
	seat := asha.Seats[0]
	if seat.RoleName != "Farmer" || seat.GroupName != "Ward A" || seat.DefinitionRoleID != "def-farmer" || seat.LeadsGroup || seat.Staff {
		t.Errorf("Asha's seat = %+v, want a Farmer at Ward A from def-farmer, leading nothing, not staff", seat)
	}
	if seat.GrantedBy != f.peter.ID || seat.GrantedByName != "Peter Mwangi" || seat.GrantedAt == "" || seat.EndedAt != "" || seat.EndedReason != "" {
		t.Errorf("Asha's seat = %+v, want granted by Peter Mwangi and still held", seat)
	}

	status, grace := standingOf(t, f.srv, f.grace.ID)
	if status != http.StatusOK {
		t.Fatalf("GET Grace's standing = %d, want 200", status)
	}
	if len(grace.Registrations) != 2 || grace.Registrations[0].GroupName != "Ward A" || grace.Registrations[1].GroupName != "Ward B" {
		t.Fatalf("Grace's registrations = %+v, want Ward A then Ward B", grace.Registrations)
	}
	if len(grace.Seats) != 1 {
		t.Fatalf("Grace's seats = %+v, want only Coordinator at Ward A and none at Ward B", grace.Seats)
	}
	seat = grace.Seats[0]
	if seat.RoleName != "Coordinator" || seat.GroupName != "Ward A" || seat.DefinitionRoleID != "def-coordinator" || !seat.LeadsGroup || !seat.Staff {
		t.Errorf("Grace's seat = %+v, want a staff Coordinator at Ward A that leads its group", seat)
	}
}

func TestAnEndedSeatStaysOnTheStandingWithItsReason(t *testing.T) {
	f := newStandingFixture(t)
	seatID := f.grantSeat(t, f.grace.ID, f.wardB.ID, f.coordinator.ID)
	if status, _ := do(t, f.srv, http.MethodPost, "/role-assignments/"+seatID+"/revoke", nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("revoke = %d", status)
	}

	_, grace := standingOf(t, f.srv, f.grace.ID)
	if len(grace.Seats) != 1 || grace.Seats[0].EndedAt == "" || grace.Seats[0].EndedReason != models.EndedByRevocation {
		t.Fatalf("Grace's seats = %+v, want the revoked Coordinator seat with its ended_at and reason", grace.Seats)
	}
}

func TestAGranterWhoIsNoActorLeavesTheNameEmptyButTheSeatListed(t *testing.T) {
	f := newStandingFixture(t)
	srv := newServer(t, routes.RoutesFor(f.mgr, routes.Mount{
		Caller: func(context.Context) string { return "operator-from-the-host" },
	}))
	f.srv = srv
	f.grantSeat(t, f.asha.ID, f.wardA.ID, f.farmer.ID)

	status, asha := standingOf(t, srv, f.asha.ID)
	if status != http.StatusOK || len(asha.Seats) != 1 {
		t.Fatalf("GET Asha's standing = %d %+v, want 200 with her one seat", status, asha)
	}
	if asha.Seats[0].GrantedBy != "operator-from-the-host" || asha.Seats[0].GrantedByName != "" {
		t.Errorf("seat = %+v, want the granter's id and no name", asha.Seats[0])
	}
}

func TestAPersonWithNothingAnswersEmptyListsNotNull(t *testing.T) {
	f := newStandingFixture(t)
	nobody, err := f.mgr.CreateActor(context.Background(), mwanachamaactor.Actor{DisplayName: "Joseph Ouma"})
	if err != nil {
		t.Fatal(err)
	}

	status, body := do(t, f.srv, http.MethodGet, "/actors/"+nobody.ID+"/standing", nil)
	registrations, regOK := body["registrations"].([]any)
	seats, seatOK := body["seats"].([]any)
	if status != http.StatusOK || !regOK || !seatOK || len(registrations) != 0 || len(seats) != 0 {
		t.Fatalf("GET standing of a person with nothing = %d %v, want 200 with [] and []", status, body)
	}
}

func TestAnUnknownActorHasNoStanding(t *testing.T) {
	f := newStandingFixture(t)

	if status, _ := standingOf(t, f.srv, f.asha.ID); status != http.StatusOK {
		t.Fatalf("GET standing of a known actor = %d, want 200", status)
	}
	if status, _ := standingOf(t, f.srv, "no-such-actor"); status != http.StatusNotFound {
		t.Errorf("GET standing of an unknown actor = %d, want 404", status)
	}
}

func TestStandingIsGatedOnItsOwnAction(t *testing.T) {
	f := newStandingFixture(t)
	gated := func(refused string) *httptest.Server {
		return newServer(t, routes.RoutesFor(f.mgr, routes.Mount{
			Caller: func(context.Context) string { return f.peter.ID },
			Authorize: func(_ context.Context, action string) error {
				if action == refused {
					return fmt.Errorf("refused")
				}
				return nil
			},
		}))
	}

	without := gated("actor.actor.standing")
	if status, _ := standingOf(t, without, f.asha.ID); status != http.StatusForbidden {
		t.Errorf("GET standing without actor.actor.standing = %d, want 403", status)
	}
	if status, _ := do(t, without, http.MethodGet, "/actors/"+f.asha.ID, nil); status != http.StatusOK {
		t.Errorf("GET the actor without actor.actor.standing = %d, want 200 — the gate is on the one action", status)
	}
	with := gated("actor.something.else")
	if status, _ := standingOf(t, with, f.asha.ID); status != http.StatusOK {
		t.Errorf("GET standing with actor.actor.standing = %d, want 200", status)
	}
}
