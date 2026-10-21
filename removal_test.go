package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestRemovingAnActorFromOneGroupKeepsTheActorAndRecordsWhoRemovedThem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Amina")
	g1 := mustGroup(t, mgr, "Ward A")
	g2 := mustGroup(t, mgr, "Ward B")
	for _, g := range []string{g1.ID, g2.ID} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}

	removal, err := mgr.RemoveFromGroup(ctx, a.ID, g1.ID, "operator-1")
	if err != nil {
		t.Fatalf("RemoveFromGroup: %v", err)
	}
	if removal.ID == "" || removal.RemovedBy != "operator-1" || removal.RemovedAt == "" || removal.GroupID != g1.ID {
		t.Errorf("removal = %+v", removal)
	}
	held, _ := mgr.ListGroupsForActor(ctx, a.ID)
	if len(held) != 1 || held[0].GroupID != g2.ID {
		t.Errorf("held after removal = %+v, want only %s", held, g2.ID)
	}
	if _, err := mgr.GetActor(ctx, a.ID); err != nil {
		t.Errorf("GetActor after removal: %v, want the actor kept", err)
	}
	if _, err := mgr.RemoveFromGroup(ctx, a.ID, g1.ID, "operator-1"); !errors.Is(err, mwanachamaactor.ErrNotInGroup) {
		t.Errorf("second removal err = %v, want ErrNotInGroup", err)
	}
}

func TestRemovingAnActorFromEveryGroupLeavesNoMembershipAndAHistoryPerGroup(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Baraka")
	g1 := mustGroup(t, mgr, "Ward A")
	g2 := mustGroup(t, mgr, "Ward B")
	for _, g := range []string{g1.ID, g2.ID} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}

	removed, err := mgr.RemoveFromAllGroups(ctx, a.ID, "operator-1")
	if err != nil || len(removed) != 2 {
		t.Fatalf("RemoveFromAllGroups = %+v, %v", removed, err)
	}
	if held, _ := mgr.ListGroupsForActor(ctx, a.ID); len(held) != 0 {
		t.Errorf("held = %+v, want none", held)
	}
	history, err := mgr.ListRemovals(ctx, a.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("ListRemovals = %+v, %v", history, err)
	}

	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g1.ID}); err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	if history, _ := mgr.ListRemovals(ctx, a.ID); len(history) != 2 {
		t.Errorf("history after rejoin = %d, want 2 kept", len(history))
	}
}

func TestARemovalMustNameWhoRemoved(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Chausiku")
	g := mustGroup(t, mgr, "Ward A")
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	if _, err := mgr.RemoveFromGroup(ctx, a.ID, g.ID, " "); !errors.Is(err, mwanachamaactor.ErrRemoverRequired) {
		t.Errorf("err = %v, want ErrRemoverRequired", err)
	}
	if held, _ := mgr.ListGroupsForActor(ctx, a.ID); len(held) != 1 {
		t.Errorf("an unnamed removal still removed: held = %+v", held)
	}
}
