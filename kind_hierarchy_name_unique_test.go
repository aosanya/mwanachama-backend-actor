package mwanachamaactor

import (
	"context"
	"errors"
	"testing"
)

func TestCreateRoleKindRefusesAnotherLiveKindWithTheSameName(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	first, err := um.CreateRoleKind(ctx, RoleKind{Name: "Probe kind"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := um.CreateRoleKind(ctx, RoleKind{Name: " PROBE  kind "}); !errors.Is(err, ErrDuplicateRoleKindName) {
		t.Fatalf("second: err = %v", err)
	}
	kinds, err := um.ListRoleKinds(ctx)
	if err != nil {
		t.Fatalf("ListRoleKinds: %v", err)
	}
	if len(kinds) != 1 || kinds[0].ID != first.ID {
		t.Fatalf("kinds = %d, want only the first", len(kinds))
	}
}

func TestACreatedRoleKindMayTakeTheNameOfARetiredKindButTheRetiredOneCannotComeBack(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	old, err := um.CreateRoleKind(ctx, RoleKind{Name: "Probe kind"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := um.RetireRoleKind(ctx, old.ID, "actor-1"); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if _, err := um.CreateRoleKind(ctx, RoleKind{Name: "Probe kind"}); err != nil {
		t.Fatalf("create under a retired name: %v", err)
	}
	if _, err := um.UnretireRoleKind(ctx, old.ID); !errors.Is(err, ErrDuplicateRoleKindName) {
		t.Fatalf("unretire onto a live name: err = %v", err)
	}
}

func TestCreateHierarchyRefusesASecondHierarchyWithTheSameName(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	first, err := um.CreateHierarchy(ctx, Hierarchy{Name: "Probe hierarchy"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := um.CreateHierarchy(ctx, Hierarchy{Name: "probe hierarchy"}); !errors.Is(err, ErrDuplicateHierarchyName) {
		t.Fatalf("second: err = %v", err)
	}
	all, err := um.ListHierarchies(ctx)
	if err != nil {
		t.Fatalf("ListHierarchies: %v", err)
	}
	if len(all) != 1 || all[0].ID != first.ID {
		t.Fatalf("hierarchies = %d, want only the first", len(all))
	}
}

func TestRenamingAHierarchyToAnotherOnesNameIsRefusedAndToItsOwnIsAdmitted(t *testing.T) {
	um, _ := newNamedManager(t)
	ctx := context.Background()
	first, err := um.CreateHierarchy(ctx, Hierarchy{Name: "Probe hierarchy"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := um.CreateHierarchy(ctx, Hierarchy{Name: "Other hierarchy"})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if _, err := um.RenameHierarchy(ctx, second.ID, "PROBE hierarchy"); !errors.Is(err, ErrDuplicateHierarchyName) {
		t.Fatalf("rename onto a taken name: err = %v", err)
	}
	if _, err := um.RenameHierarchy(ctx, first.ID, "probe hierarchy"); err != nil {
		t.Fatalf("rename to its own name: %v", err)
	}
}
