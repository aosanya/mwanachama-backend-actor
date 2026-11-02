package mwanachamaactor_test

import (
	"context"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func standingSeat(t *testing.T, seats []models.StandingSeat, id string) models.StandingSeat {
	t.Helper()
	for _, s := range seats {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("seat %s is not on the standing: %+v", id, seats)
	return models.StandingSeat{}
}

func TestStandingKeepsLiveSeatsOfDeletedAndMissingGroupsAndUnknownKinds(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)
	ctx := context.Background()
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	seatObject, _ := s.ByRole("actor_role_assignment")
	grace := mustActor(t, mgr, "Grace Adhiambo")
	closed := mustGroup(t, mgr, "Closed Ward")
	open := mustGroup(t, mgr, "Open Ward")
	chair, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Chair", LeadsGroup: true, Staff: true})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	for _, g := range []mwanachamaactor.Group{closed, open} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: grace.ID, GroupID: g.ID}); err != nil {
			t.Fatalf("AssignGroup %s: %v", g.Name, err)
		}
	}
	inClosed, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: grace.ID, GroupID: closed.ID, KindID: chair.ID})
	if err != nil {
		t.Fatalf("GrantRole at the group that is closed later: %v", err)
	}
	inOpen, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: grace.ID, GroupID: open.ID, KindID: chair.ID})
	if err != nil {
		t.Fatalf("GrantRole at the open group: %v", err)
	}
	softDelete(t, db, "group", closed.ID)
	for _, seat := range []struct{ id, groupID, kindID string }{
		{"seat-at-a-group-with-no-row", "no-such-group", chair.ID},
		{"seat-of-an-unknown-kind", open.ID, "no-such-kind"},
	} {
		if err := db.Exec(
			"insert into "+s.TableFor(seatObject)+" (id, actor_id, group_id, kind_id, active, granted_at) values (?, ?, ?, ?, ?, ?)",
			seat.id, grace.ID, seat.groupID, seat.kindID, true, "2999-01-01T00:00:00Z",
		).Error; err != nil {
			t.Fatalf("insert %s: %v", seat.id, err)
		}
	}

	standing, err := mgr.GetActorStanding(ctx, grace.ID)
	if err != nil {
		t.Fatalf("GetActorStanding: %v", err)
	}

	if len(standing.Registrations) != 1 || standing.Registrations[0].GroupID != open.ID {
		t.Errorf("registrations = %+v, want only the open group: a deleted group is dropped", standing.Registrations)
	}
	if len(standing.Seats) != 4 {
		t.Fatalf("seats = %+v, want all four live seats listed", standing.Seats)
	}
	inDeleted := standingSeat(t, standing.Seats, inClosed.ID)
	if inDeleted.EndedAt != "" || inDeleted.GroupName != "Closed Ward" || !inDeleted.LeadsGroup {
		t.Errorf("seat in the deleted group = %+v, want it still held, named Closed Ward, leading its group", inDeleted)
	}
	led, err := mgr.ListLedGroups(ctx, grace.ID)
	if err != nil {
		t.Fatalf("ListLedGroups: %v", err)
	}
	if len(led) != 1 || led[0].ID != open.ID {
		t.Errorf("led groups = %+v, want only the open group, so the standing and the led groups disagree about the deleted one", led)
	}
	if got := standingSeat(t, standing.Seats, inOpen.ID); got.GroupName != "Open Ward" || got.RoleName != "Chair" || !got.LeadsGroup {
		t.Errorf("seat in the open group = %+v, want Chair at Open Ward leading its group", got)
	}

	noGroup := standingSeat(t, standing.Seats, "seat-at-a-group-with-no-row")
	if noGroup.GroupID != "no-such-group" || noGroup.GroupName != "" || noGroup.RoleName != "Chair" {
		t.Errorf("seat at a group with no row = %+v, want its group id, an empty group_name and the Chair role", noGroup)
	}
	noKind := standingSeat(t, standing.Seats, "seat-of-an-unknown-kind")
	if noKind.KindID != "no-such-kind" || noKind.RoleName != "" || noKind.GroupName != "Open Ward" || noKind.LeadsGroup || noKind.Staff {
		t.Errorf("seat of an unknown kind = %+v, want its kind id, an empty role_name, Open Ward and no capabilities", noKind)
	}
}
