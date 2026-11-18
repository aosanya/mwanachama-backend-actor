package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

type unassignFixture struct {
	mgr          mwanachamaactor.UserManager
	ward, other  mwanachamaactor.Group
	amina, brian mwanachamaactor.Actor
	seat         mwanachamaactor.ActorRoleAssignment
}

func newUnassignFixture(t *testing.T) (unassignFixture, func(caller string) *httptest.Server) {
	t.Helper()
	mgr := newManager(t)
	ctx := context.Background()
	var f unassignFixture
	f.mgr = mgr
	var err error
	if f.ward, err = mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Water committee"}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if f.other, err = mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Roads committee"}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if f.amina, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina Hassan"}); err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if f.brian, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Brian Okello"}); err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	for _, r := range []struct{ actor, group string }{
		{f.amina.ID, f.ward.ID},
		{f.brian.ID, f.ward.ID},
		{f.amina.ID, f.other.ID},
	} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: r.actor, GroupID: r.group}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}
	if f.seat, err = mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: f.amina.ID, GroupID: f.ward.ID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	serverFor := func(caller string) *httptest.Server {
		return newServer(t, routes.RoutesFor(mgr, routes.Mount{
			CheckReferences: true,
			Caller:          func(context.Context) string { return caller },
		}))
	}
	return f, serverFor
}

func rosterActorIDs(t *testing.T, srv *httptest.Server, groupID string) []string {
	t.Helper()
	var ids []string
	for _, row := range getRoster(t, srv.URL+"/groups/"+groupID+"/assignments", "") {
		ids = append(ids, row["actor_id"].(string))
	}
	return ids
}

func hasID(ids []string, id string) bool {
	for _, got := range ids {
		if got == id {
			return true
		}
	}
	return false
}

func TestEndingARegistrationTakesThePersonOffTheGroupListAndEndsTheirSeatThere(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	base := serverFor("operator-1")
	if !hasID(rosterActorIDs(t, base, f.ward.ID), f.amina.ID) {
		t.Fatalf("before the removal Amina is not on the group list")
	}

	status, removal := do(t, base, http.MethodDelete, "/actors/"+f.amina.ID+"/assignments/"+f.ward.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE /actors/{actor_id}/assignments/{group_id} = %d %v, want 200", status, removal)
	}
	if removal["removed_by"] != "operator-1" || removal["actor_id"] != f.amina.ID || removal["group_id"] != f.ward.ID {
		t.Errorf("removal = %v, want Amina removed from the group by operator-1", removal)
	}

	ids := rosterActorIDs(t, base, f.ward.ID)
	if hasID(ids, f.amina.ID) {
		t.Errorf("group list after the removal = %v, want Amina gone", ids)
	}
	if !hasID(ids, f.brian.ID) {
		t.Errorf("group list after the removal = %v, want Brian kept", ids)
	}
	seats, err := f.mgr.ListRoleAssignmentsForGroup(context.Background(), f.ward.ID, true)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForGroup: %v", err)
	}
	for _, seat := range seats {
		if seat.ActorID == f.amina.ID {
			t.Errorf("Amina still holds seat %s in the group after the removal", seat.ID)
		}
	}
	ended, err := f.mgr.GetRoleAssignment(context.Background(), f.seat.ID)
	if err != nil || ended.Active || ended.EndedReason != mwanachamaactor.EndedByEviction {
		t.Errorf("seat after the removal = %+v, %v, want ended by eviction", ended, err)
	}
}

func TestEndingARegistrationLeavesThePersonsOtherGroupsAndRecordsTheRemoval(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	base := serverFor("operator-1")

	if status, body := do(t, base, http.MethodDelete, "/actors/"+f.amina.ID+"/assignments/"+f.ward.ID, nil); status != http.StatusOK {
		t.Fatalf("DELETE = %d %v, want 200", status, body)
	}
	if !hasID(rosterActorIDs(t, base, f.other.ID), f.amina.ID) {
		t.Errorf("Amina left the other committee too")
	}
	removals, err := f.mgr.ListRemovals(context.Background(), f.amina.ID)
	if err != nil || len(removals) != 1 || removals[0].GroupID != f.ward.ID || removals[0].RemovedBy != "operator-1" {
		t.Errorf("removals = %+v, %v, want one, from the water committee, by operator-1", removals, err)
	}
}

func TestEndingARegistrationThatIsNotThereIsNotFoundAndNeedsACaller(t *testing.T) {
	f, serverFor := newUnassignFixture(t)

	if status, body := do(t, serverFor("operator-1"), http.MethodDelete, "/actors/"+f.brian.ID+"/assignments/"+f.other.ID, nil); status != http.StatusNotFound {
		t.Errorf("removing Brian from a group he is not in = %d %v, want 404", status, body)
	}
	if status, body := do(t, serverFor(""), http.MethodDelete, "/actors/"+f.brian.ID+"/assignments/"+f.ward.ID, nil); status != http.StatusBadRequest {
		t.Errorf("removing with no caller = %d %v, want 400", status, body)
	}
	if !hasID(rosterActorIDs(t, serverFor("operator-1"), f.ward.ID), f.brian.ID) {
		t.Errorf("a refused removal took Brian off the list")
	}
}
