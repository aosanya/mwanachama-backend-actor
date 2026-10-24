package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func publishedKind(t *testing.T, mgr mwanachamaactor.UserManager, defID, name string) mwanachamaactor.RoleKind {
	t.Helper()
	k, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: name, DefinitionRoleID: defID})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	return k
}

func TestARoleKindIsFoundByItsDefinitionRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	chair := publishedKind(t, mgr, "def-chair", "Chair")
	publishedKind(t, mgr, "def-treasurer", "Treasurer")

	got, err := mgr.RoleKindForDefinitionRole(ctx, "def-chair")
	if err != nil || got.ID != chair.ID || got.DefinitionRoleID != "def-chair" {
		t.Fatalf("RoleKindForDefinitionRole = %+v, %v; want %s", got, err, chair.ID)
	}
	if _, err := mgr.RoleKindForDefinitionRole(ctx, "def-missing"); !errors.Is(err, mwanachamaactor.ErrRoleKindNotFound) {
		t.Fatalf("unknown definition role: err = %v; want ErrRoleKindNotFound", err)
	}
	if _, err := mgr.RoleKindForDefinitionRole(ctx, ""); !errors.Is(err, mwanachamaactor.ErrDefinitionRoleIDRequired) {
		t.Fatalf("empty definition role: err = %v; want ErrDefinitionRoleIDRequired", err)
	}
}

func TestSyncRenamesAKindByItsDefinitionRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	chair := publishedKind(t, mgr, "def-chair", "Chair")

	res, err := mgr.SyncDefinitionRoles(ctx, []mwanachamaactor.DefinitionRole{{ID: "def-chair", Name: "Chairperson"}}, "owner")
	if err != nil || len(res.Renamed) != 1 || len(res.Retired) != 0 {
		t.Fatalf("SyncDefinitionRoles = %+v, %v; want one rename", res, err)
	}
	got, _ := mgr.GetRoleKind(ctx, chair.ID)
	if got.Name != "Chairperson" || got.Code != chair.Code || got.Retired() {
		t.Fatalf("after rename = %+v; want Chairperson, same code, live", got)
	}
}

func TestSyncRetiresAKindDroppedFromTheDefinitionAndEndsItsSeatsAsRoleRemoved(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	chair := publishedKind(t, mgr, "def-chair", "Chair")
	treasurer := publishedKind(t, mgr, "def-treasurer", "Treasurer")
	seat, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: a.ID, GroupID: g.ID, KindID: treasurer.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}

	res, err := mgr.SyncDefinitionRoles(ctx, []mwanachamaactor.DefinitionRole{{ID: "def-chair", Name: "Chair"}}, "owner")
	if err != nil || len(res.Retired) != 1 || res.Retired[0].ID != treasurer.ID || len(res.Renamed) != 0 {
		t.Fatalf("SyncDefinitionRoles = %+v, %v; want only the treasurer retired", res, err)
	}
	got, _ := mgr.GetRoleKind(ctx, treasurer.ID)
	if !got.Retired() || got.RetiredBy != "owner" {
		t.Fatalf("dropped kind = %+v; want retired by owner", got)
	}
	kept, _ := mgr.GetRoleKind(ctx, chair.ID)
	if kept.Retired() {
		t.Fatal("a kind still in the definition was retired")
	}
	held, err := mgr.GetRoleAssignment(ctx, seat.ID)
	if err != nil || held.ActorID != a.ID || held.KindID != treasurer.ID || held.Active ||
		held.EndedReason != mwanachamaactor.EndedByRoleRemoved || held.EndedAt == "" {
		t.Fatalf("seat after retirement = %+v, %v; want it kept as history, ended role_removed", held, err)
	}
}

func TestSyncLeavesHandMadeKindsAndIsRepeatable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	hand, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Volunteer"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	publishedKind(t, mgr, "def-chair", "Chair")

	for i := 0; i < 2; i++ {
		if _, err := mgr.SyncDefinitionRoles(ctx, nil, "owner"); err != nil {
			t.Fatalf("SyncDefinitionRoles: %v", err)
		}
	}
	again, err := mgr.SyncDefinitionRoles(ctx, nil, "owner")
	if err != nil || len(again.Retired) != 0 {
		t.Fatalf("second sync = %+v, %v; want nothing retired again", again, err)
	}
	got, _ := mgr.GetRoleKind(ctx, hand.ID)
	if got.Retired() {
		t.Fatal("a hand-made kind was retired by a definition sync")
	}
	if _, err := mgr.SyncDefinitionRoles(ctx, []mwanachamaactor.DefinitionRole{{Name: "No id"}}, "owner"); !errors.Is(err, mwanachamaactor.ErrDefinitionRoleIDRequired) {
		t.Fatalf("role without id: err = %v; want ErrDefinitionRoleIDRequired", err)
	}
}

func TestADroppedLeadingRoleStopsLeadingAtOnceAndRevivesThroughUnretire(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	chair := publishedKind(t, mgr, "def-chair", "Chair")
	if _, err := mgr.SetRoleKindLeadsGroup(ctx, chair.ID, true); err != nil {
		t.Fatalf("SetRoleKindLeadsGroup: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: a.ID, GroupID: g.ID, KindID: chair.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if leads, _ := mgr.IsGroupLeader(ctx, g.ID, a.ID); !leads {
		t.Fatal("chair holder does not lead before the drop")
	}
	if _, err := mgr.SyncDefinitionRoles(ctx, nil, "owner"); err != nil {
		t.Fatalf("SyncDefinitionRoles: %v", err)
	}
	if leads, _ := mgr.IsGroupLeader(ctx, g.ID, a.ID); leads {
		t.Fatal("a dropped role still leads its group")
	}
	revived, err := mgr.UnretireRoleKind(ctx, chair.ID)
	if err != nil || revived.Retired() || revived.Code != chair.Code {
		t.Fatalf("UnretireRoleKind = %+v, %v; want the old kind live again", revived, err)
	}
	if leads, _ := mgr.IsGroupLeader(ctx, g.ID, a.ID); leads {
		t.Fatal("reviving a kind brought back a seat the drop had ended")
	}
}

func TestARetiredKindDoesNotLeadThroughAStillActiveSeat(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	chair := publishedKind(t, mgr, "def-chair", "Chair")
	if _, err := mgr.SetRoleKindLeadsGroup(ctx, chair.ID, true); err != nil {
		t.Fatalf("SetRoleKindLeadsGroup: %v", err)
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: a.ID, GroupID: g.ID, KindID: chair.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := mgr.RetireRoleKind(ctx, chair.ID, "owner"); !errors.Is(err, mwanachamaactor.ErrKindHasLiveAssignments) {
		t.Fatalf("hand retire with live seats: err = %v; want ErrKindHasLiveAssignments", err)
	}
	if leads, _ := mgr.IsGroupLeader(ctx, g.ID, a.ID); !leads {
		t.Fatal("a refused hand retire stopped the seat leading")
	}
}

func TestSyncCarriesTheStaffTickOntoTheKind(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	clerk := publishedKind(t, mgr, "def-clerk", "Clerk")
	if clerk.Staff {
		t.Fatal("a new kind is staff by default")
	}
	if _, err := mgr.SyncDefinitionRoles(ctx, []mwanachamaactor.DefinitionRole{{ID: "def-clerk", Name: "Clerk", Staff: true}}, "owner"); err != nil {
		t.Fatalf("SyncDefinitionRoles: %v", err)
	}
	got, _ := mgr.GetRoleKind(ctx, clerk.ID)
	if !got.Staff {
		t.Fatalf("after ticking staff = %+v; want staff", got)
	}
	if _, err := mgr.SyncDefinitionRoles(ctx, []mwanachamaactor.DefinitionRole{{ID: "def-clerk", Name: "Clerk"}}, "owner"); err != nil {
		t.Fatalf("SyncDefinitionRoles: %v", err)
	}
	got, _ = mgr.GetRoleKind(ctx, clerk.ID)
	if got.Staff {
		t.Fatalf("after clearing staff = %+v; want not staff", got)
	}
	created, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Inspector", Staff: true})
	if err != nil || !created.Staff {
		t.Fatalf("CreateRoleKind staff = %+v, %v", created, err)
	}
	read, _ := mgr.GetRoleKind(ctx, created.ID)
	if !read.Staff {
		t.Fatal("staff did not survive a read back")
	}
}
