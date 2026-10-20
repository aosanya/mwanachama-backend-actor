package mwanachamaactor_test

import (
	"context"
	"testing"
	"time"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func groupIDs(groups []mwanachamaactor.Group) []string {
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = g.ID
	}
	return out
}

func TestAnActorWithNoRegistrationsHasNoGroups(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4", JoinMode: models.JoinOpen}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	for _, id := range []string{a.ID, "", "  "} {
		got, err := mgr.ListMyGroups(ctx, id)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("ListMyGroups(%q) = %v, %v; want an empty list", id, got, err)
		}
	}
}

func TestMyGroupsCoverAnOpenJoinAndAnOperatorRegistrationOldestRegistrationFirst(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	operatorPlaced, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	joined, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 2", JoinMode: models.JoinOpen})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	pending, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 3"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	other, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}

	if _, err := mgr.JoinGroup(ctx, joined.ID, a.ID, ""); err != nil {
		t.Fatalf("JoinGroup open: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: operatorPlaced.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	if _, err := mgr.JoinGroup(ctx, pending.ID, a.ID, ""); err != nil {
		t.Fatalf("JoinGroup confirmed: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: other.ID, GroupID: pending.ID}); err != nil {
		t.Fatalf("AssignGroup other: %v", err)
	}

	got, err := mgr.ListMyGroups(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListMyGroups: %v", err)
	}
	ids := groupIDs(got)
	if len(ids) != 2 || ids[0] != joined.ID || ids[1] != operatorPlaced.ID {
		t.Fatalf("ListMyGroups = %v; want [%s %s], the open join before the later operator registration, without the pending request", ids, joined.ID, operatorPlaced.ID)
	}
	if got[0].Name != "Ward 2" || got[1].Name != "Ward 1" {
		t.Errorf("names = %q %q, want each group's name", got[0].Name, got[1].Name)
	}

	if _, _, err := mgr.Deregister(ctx, a.ID, joined.ID); err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	got, err = mgr.ListMyGroups(ctx, a.ID)
	if err != nil || len(got) != 1 || got[0].ID != operatorPlaced.ID {
		t.Errorf("after leaving = %v, %v; want only %s", groupIDs(got), err, operatorPlaced.ID)
	}
}
