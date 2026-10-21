package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

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

	first, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: amina.ID, GroupID: "g-1", KindID: worker.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: baraka.ID, GroupID: "g-2", KindID: worker.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: amina.ID, GroupID: "g-1", KindID: other.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}

	holders, err := mgr.ListRoleHolders(ctx, worker.ID, true)
	if err != nil {
		t.Fatalf("ListRoleHolders: %v", err)
	}
	if len(holders) != 2 || holders[0].DisplayName != "Amina" || holders[1].DisplayName != "Baraka" {
		t.Fatalf("want Amina then Baraka, got %+v", holders)
	}
	if holders[0].GroupID != "g-1" || holders[1].GroupID != "g-2" {
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

func TestListRoleHolders_AHolderWithNoActorRowStillListsWithoutAName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	k, _ := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Field worker"})
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: "ghost", GroupID: "g-1", KindID: k.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	holders, err := mgr.ListRoleHolders(ctx, k.ID, true)
	if err != nil || len(holders) != 1 || holders[0].DisplayName != "" {
		t.Fatalf("want one nameless holder, got %+v %v", holders, err)
	}
}
