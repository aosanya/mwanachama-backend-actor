package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func newTestManagerWithDB(t *testing.T) (mwanachamaactor.UserManager, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return newTestManagerOn(t, db), db
}

func softDelete(t *testing.T, db *gorm.DB, role, id string) {
	t.Helper()
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("the spec fills no object for the role %s", role)
	}
	if err := db.Exec("update "+s.TableFor(o)+" set deleted = ? where id = ?", true, id).Error; err != nil {
		t.Fatalf("soft-delete %s %s: %v", role, id, err)
	}
}

func TestGrantRoleRefusesASoftDeletedActorAndWritesNothing(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)
	ctx := context.Background()
	gone := mustActor(t, mgr, "Gone Away")
	grace := mustActor(t, mgr, "Grace Adhiambo")
	ward := mustGroup(t, mgr, "Ward A")
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	softDelete(t, db, "actor", gone.ID)

	_, err = mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: gone.ID, GroupID: ward.ID, KindID: kind.ID})
	if !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Fatalf("GrantRole to a soft-deleted actor = %v, want ErrActorNotFound", err)
	}
	seats, err := mgr.ListRoleAssignmentsForActor(ctx, gone.ID, false)
	if err != nil || len(seats) != 0 {
		t.Fatalf("seats of the soft-deleted actor = %+v, %v, want none written", seats, err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: grace.ID, GroupID: ward.ID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole to a live actor at the same group and kind = %v, want it granted", err)
	}
}

func TestGrantRoleRefusesASoftDeletedGroupAndWritesNothing(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)
	ctx := context.Background()
	grace := mustActor(t, mgr, "Grace Adhiambo")
	gone := mustGroup(t, mgr, "Closed Ward")
	ward := mustGroup(t, mgr, "Ward A")
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	softDelete(t, db, "group", gone.ID)

	_, err = mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: grace.ID, GroupID: gone.ID, KindID: kind.ID})
	if !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Fatalf("GrantRole at a soft-deleted group = %v, want ErrGroupNotFound", err)
	}
	seats, err := mgr.ListRoleAssignmentsForGroup(ctx, gone.ID, false)
	if err != nil || len(seats) != 0 {
		t.Fatalf("seats of the soft-deleted group = %+v, %v, want none written", seats, err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: grace.ID, GroupID: ward.ID, KindID: kind.ID}); err != nil {
		t.Fatalf("GrantRole to the same actor at a live group = %v, want it granted", err)
	}
}
