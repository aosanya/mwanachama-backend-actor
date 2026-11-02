package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestListRoleHolders_NamesWhoHoldsAKindAcrossGroupsAndDropsARevokedSeat(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	worker, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Field worker"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	other, _ := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"})
	amina, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	baraka, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	wardOne := mustGroup(t, mgr, "Ward One")
	wardTwo := mustGroup(t, mgr, "Ward Two")

	first, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: amina.ID, GroupID: wardOne.ID, KindID: worker.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: baraka.ID, GroupID: wardTwo.ID, KindID: worker.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: amina.ID, GroupID: wardOne.ID, KindID: other.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}

	holders, err := mgr.ListRoleHolders(ctx, worker.ID, true)
	if err != nil {
		t.Fatalf("ListRoleHolders: %v", err)
	}
	if len(holders) != 2 || holders[0].DisplayName != "Amina" || holders[1].DisplayName != "Baraka" {
		t.Fatalf("want Amina then Baraka, got %+v", holders)
	}
	if holders[0].GroupID != wardOne.ID || holders[1].GroupID != wardTwo.ID {
		t.Fatalf("each holder must carry the group the seat is in: %+v", holders)
	}

	if err := mgr.RevokeRole(ctx, first.ID); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	active, _ := mgr.ListRoleHolders(ctx, worker.ID, true)
	if len(active) != 1 || active[0].ActorID != baraka.ID {
		t.Fatalf("a revoked seat must leave the active holders, got %+v", active)
	}
	all, _ := mgr.ListRoleHolders(ctx, worker.ID, false)
	if len(all) != 2 || all[0].Active || all[0].EndedReason != "revoked" {
		t.Fatalf("history must keep the revoked seat with its reason, got %+v", all)
	}
}

func TestListRoleHolders_UnknownKindIsNotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.ListRoleHolders(context.Background(), "unknown", true); !errors.Is(err, mwanachamaactor.ErrRoleKindNotFound) {
		t.Fatalf("want ErrRoleKindNotFound, got %v", err)
	}
}

func TestListRoleHolders_AHolderWhoseActorRowIsGoneStillListsWithoutAName(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	ctx := context.Background()
	k, _ := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Field worker"})
	ward := mustGroup(t, mgr, "Ward One")
	ghost := mustActor(t, mgr, "Ghost")
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: ghost.ID, GroupID: ward.ID, KindID: k.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	actors, _ := s.ByRole("actor")
	if err := db.Exec("update "+s.TableFor(actors)+" set deleted = ? where id = ?", true, ghost.ID).Error; err != nil {
		t.Fatalf("delete the actor row: %v", err)
	}
	holders, err := mgr.ListRoleHolders(ctx, k.ID, true)
	if err != nil || len(holders) != 1 || holders[0].DisplayName != "" {
		t.Fatalf("want one nameless holder, got %+v %v", holders, err)
	}
}
