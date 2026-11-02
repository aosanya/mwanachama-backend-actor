package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

type seatFixture struct {
	mgr mwanachamaactor.UserManager
	db  *gorm.DB
	srv *httptest.Server
	ids map[string]any
}

func newSeatFixture(t *testing.T, caller string) seatFixture {
	t.Helper()
	mgr, db := newManagerWithDB(t)
	ctx := context.Background()
	actor, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Grace Adhiambo"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward A"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	srv := newServer(t, routes.RoutesFor(mgr, routes.Mount{
		Caller: func(context.Context) string { return caller },
	}))
	return seatFixture{
		mgr: mgr,
		db:  db,
		srv: srv,
		ids: map[string]any{"actor_id": actor.ID, "group_id": group.ID, "kind_id": kind.ID},
	}
}

func (f seatFixture) grant(t *testing.T, extra map[string]any) (int, map[string]any) {
	t.Helper()
	body := map[string]any{}
	for k, v := range f.ids {
		body[k] = v
	}
	for k, v := range extra {
		body[k] = v
	}
	return do(t, f.srv, http.MethodPost, "/role-assignments", body)
}

func TestAGrantedSeatRecordsTheCallerNotTheBody(t *testing.T) {
	f := newSeatFixture(t, "operator-1")

	status, seat := f.grant(t, map[string]any{"granted_by": "someone-else"})
	if status != http.StatusCreated || seat["granted_by"] != "operator-1" {
		t.Fatalf("grant naming someone-else = %d %v, want 201 recorded against operator-1", status, seat)
	}
	id, _ := seat["id"].(string)
	stored, err := f.mgr.GetRoleAssignment(context.Background(), id)
	if err != nil || stored.GrantedBy != "operator-1" {
		t.Fatalf("stored seat = %+v, %v, want granted_by operator-1", stored, err)
	}
}

func TestAGrantedSeatWithNoGrantedByStillRecordsTheCaller(t *testing.T) {
	f := newSeatFixture(t, "operator-1")

	status, seat := f.grant(t, nil)
	if status != http.StatusCreated || seat["granted_by"] != "operator-1" {
		t.Fatalf("grant with no granted_by = %d %v, want 201 recorded against operator-1", status, seat)
	}
}

func TestTheSameActorCannotHoldTheSameKindTwiceAtOneGroup(t *testing.T) {
	f := newSeatFixture(t, "operator-1")

	if status, seat := f.grant(t, nil); status != http.StatusCreated {
		t.Fatalf("first grant = %d %v, want 201", status, seat)
	}
	status, _ := f.grant(t, nil)
	if status != http.StatusConflict {
		t.Fatalf("second grant of the same seat = %d, want 409", status)
	}
	seats, err := f.mgr.ListRoleAssignmentsForActor(context.Background(), f.ids["actor_id"].(string), true)
	if err != nil || len(seats) != 1 {
		t.Fatalf("active seats = %d, %v, want 1", len(seats), err)
	}
}

func TestASeatThatEndedCanBeGrantedAgain(t *testing.T) {
	f := newSeatFixture(t, "operator-1")

	_, first := f.grant(t, nil)
	firstID, _ := first["id"].(string)
	if status, _ := do(t, f.srv, http.MethodPost, "/role-assignments/"+firstID+"/revoke", nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("revoke = %d", status)
	}
	status, second := f.grant(t, nil)
	if status != http.StatusCreated || second["id"] == firstID {
		t.Fatalf("grant after revoke = %d %v, want 201 and a new seat", status, second)
	}
	if status, _ := f.grant(t, nil); status != http.StatusConflict {
		t.Fatalf("grant over the new live seat = %d, want 409", status)
	}
}

func TestADifferentActorGroupOrKindIsNotADuplicate(t *testing.T) {
	f := newSeatFixture(t, "operator-1")
	ctx := context.Background()
	other, _ := f.mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Joseph Ouma"})
	otherGroup, _ := f.mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward B"})
	otherKind, _ := f.mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"})

	if status, _ := f.grant(t, nil); status != http.StatusCreated {
		t.Fatalf("first grant = %d", status)
	}
	for name, extra := range map[string]map[string]any{
		"another actor": {"actor_id": other.ID},
		"another group": {"group_id": otherGroup.ID},
		"another kind":  {"kind_id": otherKind.ID},
	} {
		if status, seat := f.grant(t, extra); status != http.StatusCreated {
			t.Fatalf("%s = %d %v, want 201", name, status, seat)
		}
	}
}

func TestASeatCannotBeGrantedToAnActorWhoDoesNotExist(t *testing.T) {
	f := newSeatFixture(t, "operator-1")

	if status, seat := f.grant(t, nil); status != http.StatusCreated {
		t.Fatalf("grant to a real actor = %d %v, want 201", status, seat)
	}
	status, _ := f.grant(t, map[string]any{"actor_id": "no-such-actor"})
	if status != http.StatusNotFound {
		t.Fatalf("grant to an actor who does not exist = %d, want 404", status)
	}
	seats, err := f.mgr.ListRoleAssignmentsForActor(context.Background(), "no-such-actor", false)
	if err != nil || len(seats) != 0 {
		t.Fatalf("seats of the missing actor = %+v, %v, want none written", seats, err)
	}
}

func TestASeatCannotBeGrantedAtAGroupThatDoesNotExist(t *testing.T) {
	f := newSeatFixture(t, "operator-1")

	status, _ := f.grant(t, map[string]any{"group_id": "no-such-group"})
	if status != http.StatusNotFound {
		t.Fatalf("grant at a group that does not exist = %d, want 404", status)
	}
	seats, err := f.mgr.ListRoleAssignmentsForGroup(context.Background(), "no-such-group", false)
	if err != nil || len(seats) != 0 {
		t.Fatalf("seats of the missing group = %+v, %v, want none written", seats, err)
	}
	if status, seat := f.grant(t, nil); status != http.StatusCreated {
		t.Fatalf("grant to the real actor and group = %d %v, want 201", status, seat)
	}
}

func TestASeatCannotBeGrantedToAnActorWhoIsDeleted(t *testing.T) {
	f := newSeatFixture(t, "operator-1")
	actorID, _ := f.ids["actor_id"].(string)
	softDelete(t, f.db, "actor", actorID)

	status, body := f.grant(t, nil)
	if status != http.StatusNotFound {
		t.Fatalf("grant to a deleted actor = %d %v, want 404", status, body)
	}
	seats, err := f.mgr.ListRoleAssignmentsForActor(context.Background(), actorID, false)
	if err != nil || len(seats) != 0 {
		t.Fatalf("seats of the deleted actor = %+v, %v, want none written", seats, err)
	}
}

func TestASeatCannotBeGrantedAtAGroupThatIsDeleted(t *testing.T) {
	f := newSeatFixture(t, "operator-1")
	groupID, _ := f.ids["group_id"].(string)
	softDelete(t, f.db, "group", groupID)

	status, body := f.grant(t, nil)
	if status != http.StatusNotFound {
		t.Fatalf("grant at a deleted group = %d %v, want 404", status, body)
	}
	seats, err := f.mgr.ListRoleAssignmentsForGroup(context.Background(), groupID, false)
	if err != nil || len(seats) != 0 {
		t.Fatalf("seats of the deleted group = %+v, %v, want none written", seats, err)
	}
}
