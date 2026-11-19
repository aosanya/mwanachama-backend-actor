package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	actor "github.com/aosanya/mwanachama-backend-actor"
)

type removalFixture struct {
	db     *gorm.DB
	mgr    actor.UserManager
	amina  actor.Actor
	ward   actor.Group
	seat   actor.ActorRoleAssignment
	remove string
}

func newRemovalFixture(t *testing.T) removalFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := actor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := actor.NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	ctx := context.Background()
	amina := mustActor(t, mgr, "Kovács Anna")
	ward := mustGroup(t, mgr, "Városi tanács")
	kind, err := mgr.CreateRoleKind(ctx, actor.RoleKind{Name: "Pénzügyi vezető"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, actor.ActorGroupAssignment{ActorID: amina.ID, GroupID: ward.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	seat, err := mgr.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: amina.ID, GroupID: ward.ID, KindID: kind.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return removalFixture{db: db, mgr: mgr, amina: amina, ward: ward, seat: seat, remove: "operator-1"}
}

func (f removalFixture) dropTable(t *testing.T, role string) {
	t.Helper()
	s, err := actor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("the spec fills no object for the role %s", role)
	}
	if err := f.db.Exec("drop table " + s.TableFor(o)).Error; err != nil {
		t.Fatalf("drop %s: %v", role, err)
	}
}

func (f removalFixture) registered(t *testing.T) bool {
	t.Helper()
	held, err := f.mgr.ListGroupsForActor(context.Background(), f.amina.ID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	for _, a := range held {
		if a.GroupID == f.ward.ID {
			return true
		}
	}
	return false
}

func (f removalFixture) seatNow(t *testing.T) actor.ActorRoleAssignment {
	t.Helper()
	seat, err := f.mgr.GetRoleAssignment(context.Background(), f.seat.ID)
	if err != nil {
		t.Fatalf("GetRoleAssignment: %v", err)
	}
	return seat
}

func TestAFailedRemovalRecordLeavesTheRegistrationAndSeatsAsTheyWere(t *testing.T) {
	f := newRemovalFixture(t)
	f.dropTable(t, "group_removal")

	if _, err := f.mgr.RemoveFromGroup(context.Background(), f.amina.ID, f.ward.ID, f.remove); err == nil {
		t.Fatalf("RemoveFromGroup with no removal table succeeded")
	}
	if !f.registered(t) {
		t.Errorf("Amina was taken off the list although the removal could not be recorded")
	}
	if seat := f.seatNow(t); !seat.Active {
		t.Errorf("Amina's seat ended although the removal could not be recorded: %+v", seat)
	}
}

func TestAFailedSeatUpdateLeavesTheRegistrationInPlaceSoARetryCanFinish(t *testing.T) {
	f := newRemovalFixture(t)
	f.dropTable(t, "actor_role_assignment")

	if _, err := f.mgr.RemoveFromGroup(context.Background(), f.amina.ID, f.ward.ID, f.remove); err == nil {
		t.Fatalf("RemoveFromGroup with no seat table succeeded")
	}
	if !f.registered(t) {
		t.Errorf("Amina was taken off the list although her seat could not end")
	}
	removals, err := f.mgr.ListRemovals(context.Background(), f.amina.ID)
	if err != nil {
		t.Fatalf("ListRemovals: %v", err)
	}
	if len(removals) != 0 {
		t.Errorf("removals = %d, want none", len(removals))
	}
}

func TestASeatHeldWithNoRegistrationEndsWithTheCallAndIsRecorded(t *testing.T) {
	f := newRemovalFixture(t)
	ctx := context.Background()
	if _, found, err := f.mgr.Deregister(ctx, f.amina.ID, f.ward.ID); err != nil || !found {
		t.Fatalf("Deregister = %v, %v", found, err)
	}

	if _, err := f.mgr.RemoveFromGroup(ctx, f.amina.ID, f.ward.ID, f.remove); err != nil {
		t.Fatalf("RemoveFromGroup of a seat with no registration: %v", err)
	}
	if seat := f.seatNow(t); seat.Active {
		t.Errorf("the seat is still active: %+v", seat)
	}
	removals, err := f.mgr.ListRemovals(ctx, f.amina.ID)
	if err != nil {
		t.Fatalf("ListRemovals: %v", err)
	}
	if len(removals) != 1 {
		t.Errorf("removals = %d, want 1", len(removals))
	}
}

func TestRemovingTheSamePersonTwiceIsRefusedAndRecordsOneRemoval(t *testing.T) {
	f := newRemovalFixture(t)
	ctx := context.Background()
	if _, err := f.mgr.RemoveFromGroup(ctx, f.amina.ID, f.ward.ID, f.remove); err != nil {
		t.Fatalf("first RemoveFromGroup: %v", err)
	}

	if _, err := f.mgr.RemoveFromGroup(ctx, f.amina.ID, f.ward.ID, f.remove); !errors.Is(err, actor.ErrNotInGroup) {
		t.Errorf("second RemoveFromGroup = %v, want ErrNotInGroup", err)
	}
	removals, err := f.mgr.ListRemovals(ctx, f.amina.ID)
	if err != nil {
		t.Fatalf("ListRemovals: %v", err)
	}
	if len(removals) != 1 {
		t.Errorf("removals = %d, want 1", len(removals))
	}
}
