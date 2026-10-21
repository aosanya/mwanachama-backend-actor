package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func priorityOf(t *testing.T, mgr mwanachamaactor.UserManager, actorID, groupID string) string {
	t.Helper()
	groups, err := mgr.ListMyGroups(context.Background(), actorID)
	if err != nil {
		t.Fatalf("ListMyGroups: %v", err)
	}
	for _, g := range groups {
		if g.ID == groupID {
			return g.Priority
		}
	}
	t.Fatalf("group %s not among %s's groups", groupID, actorID)
	return ""
}

func TestEveryRegistrationStartsAtNormalPriorityAJoinedOneToo(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	placed, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	open, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 2", JoinMode: models.JoinOpen})
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	assigned, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: placed.ID})
	if err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	if assigned.Priority != models.PriorityNormal {
		t.Errorf("AssignGroup priority = %q, want normal", assigned.Priority)
	}
	if _, err := mgr.JoinGroup(ctx, open.ID, a.ID, ""); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	for _, g := range []string{placed.ID, open.ID} {
		if got := priorityOf(t, mgr, a.ID, g); got != models.PriorityNormal {
			t.Errorf("priority of %s = %q, want normal", g, got)
		}
	}
}

func TestAMemberSetsTheirOwnPriorityAndTheGroupsOfMineReadReturnsIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	a, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	b, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	for _, who := range []string{a.ID, b.ID} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: who, GroupID: g.ID}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}
	for _, p := range []string{models.PriorityVeryHigh, models.PriorityHigh, models.PriorityLow, models.PriorityVeryLow, models.PriorityNormal, models.PriorityVeryHigh} {
		got, err := mgr.SetMyGroupPriority(ctx, g.ID, a.ID, p)
		if err != nil || got.Priority != p {
			t.Fatalf("SetMyGroupPriority(%q) = %+v, %v", p, got, err)
		}
		if read := priorityOf(t, mgr, a.ID, g.ID); read != p {
			t.Errorf("read priority = %q, want %q", read, p)
		}
	}
	if got := priorityOf(t, mgr, b.ID, g.ID); got != models.PriorityNormal {
		t.Errorf("another member's priority = %q, want normal untouched", got)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
		t.Fatalf("re-AssignGroup: %v", err)
	}
	if got := priorityOf(t, mgr, a.ID, g.ID); got != models.PriorityVeryHigh {
		t.Errorf("priority after a re-assign = %q, want very_high kept", got)
	}
}

func TestSettingAPriorityRefusesAnUnknownValueAndAGroupNotHeld(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	other, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 2"})
	a, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	for _, p := range []string{"", "urgent", "Normal"} {
		if _, err := mgr.SetMyGroupPriority(ctx, g.ID, a.ID, p); !errors.Is(err, mwanachamaactor.ErrInvalidPriority) {
			t.Errorf("SetMyGroupPriority(%q) err = %v, want ErrInvalidPriority", p, err)
		}
	}
	for _, gid := range []string{other.ID, "no-such-group"} {
		if _, err := mgr.SetMyGroupPriority(ctx, gid, a.ID, models.PriorityHigh); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
			t.Errorf("SetMyGroupPriority(%s) err = %v, want ErrGroupNotFound", gid, err)
		}
	}
	if _, err := mgr.SetMyGroupPriority(ctx, g.ID, "", models.PriorityHigh); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Errorf("no caller err = %v, want ErrGroupNotFound", err)
	}
}
