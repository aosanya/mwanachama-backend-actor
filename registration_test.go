package mwanachamaactor_test

import (
	"context"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func mustActor(t *testing.T, mgr mwanachamaactor.UserManager, name string) models.Actor {
	t.Helper()
	a, err := mgr.CreateActor(context.Background(), models.Actor{DisplayName: name})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	return a
}

func mustGroup(t *testing.T, mgr mwanachamaactor.UserManager, name string) models.Group {
	t.Helper()
	g, err := mgr.CreateGroup(context.Background(), models.Group{Name: name})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return g
}

func TestRegister_RoundTrips(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Amina")
	g := mustGroup(t, mgr, "Ward A")

	reg, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{
		ActorID: a.ID, GroupID: g.ID, IsHome: true,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !reg.IsHome || reg.JoinedAt == "" {
		t.Errorf("Register result = %+v", reg)
	}

	regs, err := mgr.ListGroupsForActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	if len(regs) != 1 || regs[0].GroupID != g.ID {
		t.Errorf("ListGroupsForActor = %v", regs)
	}
}

func TestRegister_MissingActor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g := mustGroup(t, mgr, "Ward A")

	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: "nope", GroupID: g.ID}); err == nil {
		t.Fatal("expected an error registering a nonexistent actor")
	}
}

// TestRegister_NoHomeExclusivity is the DSN-1698 decision-8 parity check:
// registering a SECOND home group must NOT clear the first one's IsHome
// flag. This replaces the old "one home chapter per member" DB-conflict
// assertion the gateway's pre-cutover tests made — see this repo's
// documentation/2. design/README.md and the gateway board's DSN-1698 note.
func TestRegister_NoHomeExclusivity(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Amina")
	g1 := mustGroup(t, mgr, "Ward A")
	g2 := mustGroup(t, mgr, "Ward B")

	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a.ID, GroupID: g1.ID, IsHome: true}); err != nil {
		t.Fatalf("Register g1: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a.ID, GroupID: g2.ID, IsHome: true}); err != nil {
		t.Fatalf("Register g2: %v", err)
	}

	regs, err := mgr.ListGroupsForActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	homeCount := 0
	for _, r := range regs {
		if r.IsHome {
			homeCount++
		}
	}
	if homeCount != 2 {
		t.Fatalf("homeCount = %d, want 2 (no exclusivity enforced)", homeCount)
	}
}

func TestRegister_ReRegisterPreservesJoinedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Amina")
	g := mustGroup(t, mgr, "Ward A")

	first, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{
		ActorID: a.ID, GroupID: g.ID, IsHome: false, JoinedAt: "2020-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}
	second, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{
		ActorID: a.ID, GroupID: g.ID, IsHome: true,
	})
	if err != nil {
		t.Fatalf("second Register: %v", err)
	}
	if second.JoinedAt != first.JoinedAt {
		t.Errorf("JoinedAt changed on re-register: %q -> %q", first.JoinedAt, second.JoinedAt)
	}
	if !second.IsHome {
		t.Error("expected IsHome updated to true")
	}

	regs, err := mgr.ListGroupsForActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	if len(regs) != 1 {
		t.Fatalf("len(regs) = %d, want 1 (re-register must not duplicate)", len(regs))
	}
}

func TestDeregister_RemovesEdgeAndReturnsPriorState(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Amina")
	g := mustGroup(t, mgr, "Ward A")

	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID, IsHome: true}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	gone, found, err := mgr.Deregister(ctx, a.ID, g.ID)
	if err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if !gone.IsHome {
		t.Error("expected the removed ActorGroupAssignment to carry IsHome=true")
	}

	regs, err := mgr.ListGroupsForActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	if len(regs) != 0 {
		t.Errorf("regs = %v, want empty after Deregister", regs)
	}
}

func TestDeregister_IdempotentNoOp(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a := mustActor(t, mgr, "Amina")
	g := mustGroup(t, mgr, "Ward A")

	_, found, err := mgr.Deregister(ctx, a.ID, g.ID)
	if err != nil {
		t.Fatalf("Deregister on never-registered pair: %v", err)
	}
	if found {
		t.Fatal("expected found=false for a pair never registered")
	}
}

func TestListActorsForGroup(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g := mustGroup(t, mgr, "Ward A")
	a1 := mustActor(t, mgr, "Amina")
	a2 := mustActor(t, mgr, "Baraka")

	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a1.ID, GroupID: g.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a2.ID, GroupID: g.ID}); err != nil {
		t.Fatal(err)
	}

	regs, err := mgr.ListActorsForGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListActorsForGroup: %v", err)
	}
	if len(regs) != 2 {
		t.Fatalf("len(regs) = %d, want 2", len(regs))
	}
}

func TestHomeCounts_OnlyCountsHome(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g1 := mustGroup(t, mgr, "Ward A")
	g2 := mustGroup(t, mgr, "Ward B")
	a1 := mustActor(t, mgr, "Amina")
	a2 := mustActor(t, mgr, "Baraka")
	a3 := mustActor(t, mgr, "Chiku")

	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a1.ID, GroupID: g1.ID, IsHome: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a2.ID, GroupID: g1.ID, IsHome: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.AssignGroup(ctx, models.ActorGroupAssignment{ActorID: a3.ID, GroupID: g2.ID, IsHome: true}); err != nil {
		t.Fatal(err)
	}

	counts, err := mgr.HomeCounts(ctx)
	if err != nil {
		t.Fatalf("HomeCounts: %v", err)
	}
	if counts[g1.ID] != 1 {
		t.Errorf("counts[g1] = %d, want 1", counts[g1.ID])
	}
	if counts[g2.ID] != 1 {
		t.Errorf("counts[g2] = %d, want 1", counts[g2.ID])
	}
	if _, ok := counts["no-such-group"]; ok {
		t.Error("groups with nobody should be absent, not zero")
	}
}
