package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestRoleKindLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	k, err := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "Coordinator", Capabilities: []string{"post_chat"}})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if k.ID == "" {
		t.Fatal("expected minted kind id")
	}
	if k.RetiredAt != "" || k.RetiredBy != "" {
		t.Fatalf("a new kind must not be born retired: %+v", k)
	}

	got, err := mgr.GetRoleKind(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetRoleKind: %v", err)
	}
	if got.Name != "Coordinator" || len(got.Capabilities) != 1 {
		t.Fatalf("wrong kind: %+v", got)
	}

	if _, err := mgr.GetRoleKind(ctx, "unknown"); !errors.Is(err, mwanachamaactor.ErrRoleKindNotFound) {
		t.Fatalf("expected ErrRoleKindNotFound, got %v", err)
	}

	noCaps, err := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "nocaps"})
	if err != nil {
		t.Fatalf("CreateRoleKind(nocaps): %v", err)
	}
	if noCaps.Capabilities == nil {
		t.Fatal("DEV-1077: a kind with no capabilities must come back as [], not nil")
	}

	all, err := mgr.ListRoleKinds(ctx)
	if err != nil {
		t.Fatalf("ListRoleKinds: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 kinds, got %d", len(all))
	}
}

func TestRetireRefusedWhileAssignmentsLive(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	k, _ := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "Coordinator"})
	a, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-1", GroupID: "g-1", KindID: k.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}

	if _, err := mgr.RetireRoleKind(ctx, k.ID, "hq-admin"); !errors.Is(err, mwanachamaactor.ErrKindHasLiveAssignments) {
		t.Fatalf("expected ErrKindHasLiveAssignments, got %v", err)
	}

	if err := mgr.RevokeRole(ctx, a.ID); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	retired, err := mgr.RetireRoleKind(ctx, k.ID, "hq-admin")
	if err != nil {
		t.Fatalf("RetireRoleKind after revoke: %v", err)
	}
	if retired.RetiredAt == "" || retired.RetiredBy != "hq-admin" {
		t.Fatalf("retirement did not stamp: %+v", retired)
	}

	// Idempotent: a second retire keeps the original stamp.
	second, err := mgr.RetireRoleKind(ctx, k.ID, "somebody-else")
	if err != nil {
		t.Fatalf("second retire: %v", err)
	}
	if second.RetiredBy != "hq-admin" {
		t.Fatalf("second retire overwrote the original actor: %+v", second)
	}

	// Grant onto a retired kind is refused (DEV-1201).
	if _, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-2", GroupID: "g-1", KindID: k.ID}); !errors.Is(err, mwanachamaactor.ErrKindRetired) {
		t.Fatalf("expected ErrKindRetired, got %v", err)
	}

	back, err := mgr.UnretireRoleKind(ctx, k.ID)
	if err != nil {
		t.Fatalf("UnretireRoleKind: %v", err)
	}
	if back.RetiredAt != "" || back.RetiredBy != "" {
		t.Fatalf("unretire left an ending behind: %+v", back)
	}
	if _, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-2", GroupID: "g-1", KindID: k.ID}); err != nil {
		t.Fatalf("grant after unretire: %v", err)
	}
}

func TestRoleGrantRevokeStepDownAndListing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	k, _ := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "Coordinator"})
	a1, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-1", GroupID: "g-1", KindID: k.ID})
	if err != nil {
		t.Fatalf("grant a1: %v", err)
	}
	if !a1.Active || a1.GrantedAt == "" {
		t.Fatalf("expected grant to be active with stamped granted_at, got %+v", a1)
	}
	a2, _ := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-2", GroupID: "g-1", KindID: k.ID})
	_, _ = mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-3", GroupID: "g-other", KindID: k.ID})

	live, err := mgr.ListRoleAssignmentsForGroup(ctx, "g-1", true)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForGroup: %v", err)
	}
	if len(live) != 2 {
		t.Fatalf("expected 2 active at g-1, got %d", len(live))
	}

	if err := mgr.RevokeRole(ctx, a1.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := mgr.StepDownRole(ctx, a2.ID); err != nil {
		t.Fatalf("step down: %v", err)
	}

	live, _ = mgr.ListRoleAssignmentsForGroup(ctx, "g-1", true)
	if len(live) != 0 {
		t.Fatalf("expected 0 active after revoke + step-down, got %d", len(live))
	}
	all, _ := mgr.ListRoleAssignmentsForGroup(ctx, "g-1", false)
	if len(all) != 2 {
		t.Fatalf("expected both records still present when activeOnly=false, got %d", len(all))
	}

	ended1, err := mgr.GetRoleAssignment(ctx, a1.ID)
	if err != nil {
		t.Fatalf("GetRoleAssignment a1: %v", err)
	}
	if ended1.EndedReason != models.EndedByRevocation {
		t.Errorf("a1 ended reason = %q, want revoked", ended1.EndedReason)
	}
	ended2, err := mgr.GetRoleAssignment(ctx, a2.ID)
	if err != nil {
		t.Fatalf("GetRoleAssignment a2: %v", err)
	}
	if ended2.EndedReason != models.EndedByResignation {
		t.Errorf("a2 ended reason = %q, want resigned", ended2.EndedReason)
	}

	if err := mgr.RevokeRole(ctx, "unknown"); !errors.Is(err, mwanachamaactor.ErrAssignmentNotFound) {
		t.Fatalf("expected ErrAssignmentNotFound for unknown revoke, got %v", err)
	}

	// Re-ending an already-ended seat keeps the first reason.
	if err := mgr.RevokeRole(ctx, a2.ID); err != nil {
		t.Fatalf("re-revoke of a stepped-down seat should be a no-op, got %v", err)
	}
	still, _ := mgr.GetRoleAssignment(ctx, a2.ID)
	if still.EndedReason != models.EndedByResignation {
		t.Fatalf("a later revoke rewrote the reason: %q", still.EndedReason)
	}
}

func TestRoleAssignmentsForActorAcrossGroups(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	k1, _ := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "Coordinator"})
	k2, _ := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "Organizer"})
	if _, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-1", GroupID: "g-1", KindID: k1.ID}); err != nil {
		t.Fatalf("grant 1: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-1", GroupID: "g-2", KindID: k2.ID}); err != nil {
		t.Fatalf("grant 2: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-2", GroupID: "g-1", KindID: k1.ID}); err != nil {
		t.Fatalf("grant 3: %v", err)
	}

	mine, err := mgr.ListRoleAssignmentsForActor(ctx, "m-1", true)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForActor: %v", err)
	}
	if len(mine) != 2 {
		t.Fatalf("expected 2 assignments for m-1 across groups, got %d", len(mine))
	}
}

func TestEndOnEvictionAndDeparture(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	k, _ := mgr.CreateRoleKind(ctx, models.RoleKind{Name: "Coordinator"})
	a1, _ := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-1", GroupID: "g-1", KindID: k.ID})
	a2, _ := mgr.GrantRole(ctx, models.ActorRoleAssignment{ActorID: "m-2", GroupID: "g-1", KindID: k.ID})

	if err := mgr.EndRoleOnEviction(ctx, a1.ID); err != nil {
		t.Fatalf("EndRoleOnEviction: %v", err)
	}
	if err := mgr.EndRoleOnDeparture(ctx, a2.ID); err != nil {
		t.Fatalf("EndRoleOnDeparture: %v", err)
	}
	got1, _ := mgr.GetRoleAssignment(ctx, a1.ID)
	if got1.Active || got1.EndedReason != models.EndedByEviction {
		t.Fatalf("a1 = %+v, want inactive/evicted", got1)
	}
	got2, _ := mgr.GetRoleAssignment(ctx, a2.ID)
	if got2.Active || got2.EndedReason != models.EndedByDeparture {
		t.Fatalf("a2 = %+v, want inactive/deregistered", got2)
	}
}
