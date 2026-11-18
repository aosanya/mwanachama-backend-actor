package routes_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

type unassignFixture struct {
	mgr                 mwanachamaactor.UserManager
	db                  *gorm.DB
	ward, other         mwanachamaactor.Group
	amina, brian, chuma mwanachamaactor.Actor
	treasurer, auditor  mwanachamaactor.RoleKind
	aminaSeat           mwanachamaactor.ActorRoleAssignment
	aminaOtherSeat      mwanachamaactor.ActorRoleAssignment
	brianSeat           mwanachamaactor.ActorRoleAssignment
}

func newUnassignFixture(t *testing.T) (unassignFixture, func(caller string) *httptest.Server) {
	t.Helper()
	mgr, db := newManagerWithDB(t)
	ctx := context.Background()
	f := unassignFixture{mgr: mgr, db: db}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	var err error
	f.ward, err = mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Water committee"})
	must(err)
	f.other, err = mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Roads committee"})
	must(err)
	f.amina, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina Hassan"})
	must(err)
	f.brian, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Brian Okello"})
	must(err)
	f.chuma, err = mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Chuma Eze"})
	must(err)
	f.treasurer, err = mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"})
	must(err)
	f.auditor, err = mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Auditor"})
	must(err)
	for _, r := range []struct{ actor, group string }{
		{f.amina.ID, f.ward.ID},
		{f.brian.ID, f.ward.ID},
		{f.amina.ID, f.other.ID},
	} {
		_, err = mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: r.actor, GroupID: r.group})
		must(err)
	}
	f.aminaSeat, err = mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: f.amina.ID, GroupID: f.ward.ID, KindID: f.treasurer.ID})
	must(err)
	f.aminaOtherSeat, err = mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: f.amina.ID, GroupID: f.other.ID, KindID: f.treasurer.ID})
	must(err)
	f.brianSeat, err = mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: f.brian.ID, GroupID: f.ward.ID, KindID: f.treasurer.ID})
	must(err)
	serverFor := func(caller string) *httptest.Server {
		return newServer(t, routes.RoutesFor(mgr, routes.Mount{
			CheckReferences: true,
			Caller:          func(context.Context) string { return caller },
		}))
	}
	return f, serverFor
}

func (f unassignFixture) path(actorID, groupID string) string {
	return "/actors/" + actorID + "/assignments/" + groupID
}

func (f unassignFixture) seat(t *testing.T, id string) mwanachamaactor.ActorRoleAssignment {
	t.Helper()
	seat, err := f.mgr.GetRoleAssignment(context.Background(), id)
	if err != nil {
		t.Fatalf("GetRoleAssignment %s: %v", id, err)
	}
	return seat
}

func (f unassignFixture) removals(t *testing.T, actorID string) int {
	t.Helper()
	removals, err := f.mgr.ListRemovals(context.Background(), actorID)
	if err != nil {
		t.Fatalf("ListRemovals: %v", err)
	}
	return len(removals)
}

func (f unassignFixture) registered(t *testing.T, actorID, groupID string) bool {
	t.Helper()
	held, err := f.mgr.ListGroupsForActor(context.Background(), actorID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	for _, a := range held {
		if a.GroupID == groupID {
			return true
		}
	}
	return false
}

func dropTable(t *testing.T, db *gorm.DB, role string) {
	t.Helper()
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("the spec fills no object for the role %s", role)
	}
	if err := db.Exec("drop table " + s.TableFor(o)).Error; err != nil {
		t.Fatalf("drop %s: %v", role, err)
	}
}

func rawDelete(t *testing.T, srv *httptest.Server, path string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
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

func TestTheUnassignToolIsDeclaredUnderItsOwnAction(t *testing.T) {
	tools, err := routes.BuildTools(newManager(t))
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == "actor_unassign_group" {
			if tool.Action != "actor.assignment.unassign" {
				t.Errorf("actor_unassign_group carries action %q, want actor.assignment.unassign", tool.Action)
			}
			return
		}
	}
	t.Errorf("no tool named actor_unassign_group is published")
}

func TestEndingARegistrationTakesThePersonOffTheGroupListAndEndsTheirSeatThere(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	srv := serverFor("operator-1")
	if !hasID(rosterActorIDs(t, srv, f.ward.ID), f.amina.ID) {
		t.Fatalf("before the removal Amina is not on the group list")
	}

	status, removal := do(t, srv, http.MethodDelete, f.path(f.amina.ID, f.ward.ID), nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE /actors/{actor_id}/assignments/{group_id} = %d %v, want 200", status, removal)
	}
	if removal["removed_by"] != "operator-1" || removal["actor_id"] != f.amina.ID || removal["group_id"] != f.ward.ID {
		t.Errorf("removal = %v, want Amina removed from the group by operator-1", removal)
	}

	ids := rosterActorIDs(t, srv, f.ward.ID)
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
	ended := f.seat(t, f.aminaSeat.ID)
	if ended.Active || ended.EndedReason != mwanachamaactor.EndedByEviction || ended.EndedAt == "" {
		t.Errorf("seat after the removal = %+v, want ended by eviction with a time", ended)
	}
}

func TestEndingARegistrationLeavesEverythingElseAlone(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	srv := serverFor("operator-1")

	if status, body := do(t, srv, http.MethodDelete, f.path(f.amina.ID, f.ward.ID), nil); status != http.StatusOK {
		t.Fatalf("DELETE = %d %v, want 200", status, body)
	}
	if !hasID(rosterActorIDs(t, srv, f.other.ID), f.amina.ID) {
		t.Errorf("Amina left the other committee too")
	}
	if seat := f.seat(t, f.aminaOtherSeat.ID); !seat.Active || seat.EndedReason != "" {
		t.Errorf("Amina's seat in the other group = %+v, want still active", seat)
	}
	if seat := f.seat(t, f.brianSeat.ID); !seat.Active || seat.EndedReason != "" {
		t.Errorf("Brian's seat in the same group = %+v, want still active", seat)
	}
	if _, err := f.mgr.GetActor(context.Background(), f.amina.ID); err != nil {
		t.Errorf("GetActor after the removal: %v, want the actor record kept", err)
	}
	removals, err := f.mgr.ListRemovals(context.Background(), f.amina.ID)
	if err != nil || len(removals) != 1 || removals[0].GroupID != f.ward.ID || removals[0].RemovedBy != "operator-1" {
		t.Errorf("removals = %+v, %v, want one, from the water committee, by operator-1", removals, err)
	}
}

func TestEndingARegistrationKeepsTheReasonAndTimeOfASeatThatAlreadyEnded(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	ctx := context.Background()
	earlier, err := f.mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: f.amina.ID, GroupID: f.ward.ID, KindID: f.auditor.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if err := f.mgr.RevokeRole(ctx, earlier.ID); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	before := f.seat(t, earlier.ID)

	if status, body := do(t, serverFor("operator-1"), http.MethodDelete, f.path(f.amina.ID, f.ward.ID), nil); status != http.StatusOK {
		t.Fatalf("DELETE = %d %v, want 200", status, body)
	}
	after := f.seat(t, earlier.ID)
	if after.EndedReason != before.EndedReason || after.EndedAt != before.EndedAt {
		t.Errorf("seat ended before the removal = %q at %q after it, was %q at %q", after.EndedReason, after.EndedAt, before.EndedReason, before.EndedAt)
	}
	if after.EndedReason == mwanachamaactor.EndedByEviction {
		t.Errorf("a seat that was revoked reads as an eviction")
	}
}

func TestEndingTheSameRegistrationTwiceIsNotFoundAndRecordsOneRemoval(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	srv := serverFor("operator-1")

	if status, body := do(t, srv, http.MethodDelete, f.path(f.amina.ID, f.ward.ID), nil); status != http.StatusOK {
		t.Fatalf("first DELETE = %d %v, want 200", status, body)
	}
	status, body := rawDelete(t, srv, f.path(f.amina.ID, f.ward.ID))
	if status != http.StatusNotFound || !strings.Contains(body, "not registered") {
		t.Errorf("second DELETE = %d %q, want 404 saying the actor is not registered", status, body)
	}
	if n := f.removals(t, f.amina.ID); n != 1 {
		t.Errorf("removals after two calls = %d, want 1", n)
	}
}

func TestEndingARegistrationOfSomeoneOrSomewhereUnknownIsNotFoundAndWritesNothing(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	srv := serverFor("operator-1")

	for name, path := range map[string]string{
		"unknown person":       f.path("no-such-person", f.ward.ID),
		"unknown group":        f.path(f.amina.ID, "no-such-group"),
		"person not in group":  f.path(f.chuma.ID, f.ward.ID),
		"other group not held": f.path(f.brian.ID, f.other.ID),
	} {
		status, body := rawDelete(t, srv, path)
		if status != http.StatusNotFound || !strings.Contains(body, "not registered") {
			t.Errorf("%s: DELETE = %d %q, want 404 saying the actor is not registered", name, status, body)
		}
	}
	for _, id := range []string{f.amina.ID, f.brian.ID, f.chuma.ID, "no-such-person"} {
		if n := f.removals(t, id); n != 0 {
			t.Errorf("removals for %s = %d, want none", id, n)
		}
	}
	if !f.registered(t, f.amina.ID, f.ward.ID) || !f.seat(t, f.aminaSeat.ID).Active {
		t.Errorf("a refused call changed Amina's registration or seat")
	}
}

func TestEndingARegistrationWithNoCallerIsRefusedAndWritesNothing(t *testing.T) {
	f, serverFor := newUnassignFixture(t)

	status, body := rawDelete(t, serverFor(""), f.path(f.amina.ID, f.ward.ID))
	if status != http.StatusBadRequest || !strings.Contains(body, "names who removed") {
		t.Errorf("removing with no caller = %d %q, want 400 saying a removal names who removed the actor", status, body)
	}
	if n := f.removals(t, f.amina.ID); n != 0 {
		t.Errorf("removals after the refused call = %d, want none", n)
	}
	if !f.registered(t, f.amina.ID, f.ward.ID) || !f.seat(t, f.aminaSeat.ID).Active {
		t.Errorf("the refused call took Amina off the list or ended her seat")
	}
}

func declaredUnassignAction(t *testing.T) string {
	t.Helper()
	tools, err := routes.BuildTools(newManager(t))
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == "actor_unassign_group" {
			return tool.Action
		}
	}
	t.Fatalf("no tool named actor_unassign_group is published")
	return ""
}

func TestEndingARegistrationIsRefusedWhenTheMountRefusesTheAction(t *testing.T) {
	f, _ := newUnassignFixture(t)
	var asked []string
	srv := newServer(t, routes.RoutesFor(f.mgr, routes.Mount{
		CheckReferences: true,
		Caller:          func(context.Context) string { return "operator-1" },
		Authorize: func(_ context.Context, action string) error {
			asked = append(asked, action)
			return dispatch.ErrForbidden
		},
	}))

	status, body := rawDelete(t, srv, f.path(f.amina.ID, f.ward.ID))
	if status != http.StatusForbidden {
		t.Errorf("DELETE with a refusing mount = %d %q, want 403", status, body)
	}
	declared := declaredUnassignAction(t)
	if len(asked) != 1 || asked[0] != declared {
		t.Errorf("the mount was asked about %v, want %s once", asked, declared)
	}
	if n := f.removals(t, f.amina.ID); n != 0 || !f.registered(t, f.amina.ID, f.ward.ID) || !f.seat(t, f.aminaSeat.ID).Active {
		t.Errorf("a refused call changed Amina's registration, seat or history")
	}
}

func TestASeatHeldWithNoRegistrationEndsWithTheCallAndIsRecorded(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	ctx := context.Background()
	seat, err := f.mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: f.chuma.ID, GroupID: f.ward.ID, KindID: f.treasurer.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	srv := serverFor("operator-1")

	if status, body := do(t, srv, http.MethodDelete, f.path(f.chuma.ID, f.ward.ID), nil); status != http.StatusOK {
		t.Fatalf("DELETE for a seat holder with no registration = %d %v, want 200", status, body)
	}
	if ended := f.seat(t, seat.ID); ended.Active || ended.EndedReason != mwanachamaactor.EndedByEviction {
		t.Errorf("seat = %+v, want ended by eviction", ended)
	}
	if n := f.removals(t, f.chuma.ID); n != 1 {
		t.Errorf("removals = %d, want 1", n)
	}
	if status, _ := rawDelete(t, srv, f.path(f.chuma.ID, f.ward.ID)); status != http.StatusNotFound {
		t.Errorf("second DELETE = %d, want 404", status)
	}
}

func TestAFailedRemovalRecordLeavesTheRegistrationAndSeatsAsTheyWere(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	dropTable(t, f.db, "group_removal")

	status, body := rawDelete(t, serverFor("operator-1"), f.path(f.amina.ID, f.ward.ID))
	if status < 500 {
		t.Fatalf("DELETE with no removal table = %d %q, want a server error", status, body)
	}
	if !f.registered(t, f.amina.ID, f.ward.ID) {
		t.Errorf("Amina was taken off the list although the removal could not be recorded")
	}
	if seat := f.seat(t, f.aminaSeat.ID); !seat.Active {
		t.Errorf("Amina's seat ended although the removal could not be recorded: %+v", seat)
	}
}

func TestAFailedSeatUpdateLeavesTheRegistrationInPlaceSoARetryCanFinish(t *testing.T) {
	f, serverFor := newUnassignFixture(t)
	dropTable(t, f.db, "actor_role_assignment")

	status, body := rawDelete(t, serverFor("operator-1"), f.path(f.amina.ID, f.ward.ID))
	if status < 500 {
		t.Fatalf("DELETE with no seat table = %d %q, want a server error", status, body)
	}
	if !f.registered(t, f.amina.ID, f.ward.ID) {
		t.Errorf("Amina was taken off the list although her seat could not end")
	}
	if n := f.removals(t, f.amina.ID); n != 0 {
		t.Errorf("removals = %d, want none", n)
	}
}
