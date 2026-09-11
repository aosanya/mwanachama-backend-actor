package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestHierarchyLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Kenya national council"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	if h.ID == "" {
		t.Fatal("expected minted hierarchy id")
	}

	got, err := mgr.GetHierarchy(ctx, h.ID)
	if err != nil {
		t.Fatalf("GetHierarchy: %v", err)
	}
	if got.Name != "Kenya national council" {
		t.Fatalf("wrong hierarchy: %+v", got)
	}

	if _, err := mgr.GetHierarchy(ctx, "unknown"); !errors.Is(err, mwanachamaactor.ErrHierarchyNotFound) {
		t.Fatalf("expected ErrHierarchyNotFound, got %v", err)
	}

	renamed, err := mgr.RenameHierarchy(ctx, h.ID, "Kenya national council v2")
	if err != nil {
		t.Fatalf("RenameHierarchy: %v", err)
	}
	if renamed.Name != "Kenya national council v2" {
		t.Fatalf("rename did not stick: %+v", renamed)
	}

	if _, err := mgr.RenameHierarchy(ctx, "unknown", "x"); !errors.Is(err, mwanachamaactor.ErrHierarchyNotFound) {
		t.Fatalf("expected ErrHierarchyNotFound on rename, got %v", err)
	}

	if _, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Second"}); err != nil {
		t.Fatalf("CreateHierarchy(second): %v", err)
	}
	all, err := mgr.ListHierarchies(ctx)
	if err != nil {
		t.Fatalf("ListHierarchies: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 hierarchies, got %d", len(all))
	}
}

func TestCreateHierarchyAndLevel_CodeIsSequentialAndImmutable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h1, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Kenya"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	h2, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Uganda"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	if h1.Code != "H-1" || h2.Code != "H-2" {
		t.Fatalf("Codes = %q, %q, want H-1, H-2 (sequential across repeated Creates)", h1.Code, h2.Code)
	}

	// Code survives RenameHierarchy unchanged — that call writes name only.
	renamed, err := mgr.RenameHierarchy(ctx, h1.ID, "Kenya v2")
	if err != nil {
		t.Fatalf("RenameHierarchy: %v", err)
	}
	if renamed.Code != h1.Code {
		t.Fatalf("Code changed after rename: got %q, want %q", renamed.Code, h1.Code)
	}

	l1, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h1.ID, Name: "Ward"})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	l2, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h1.ID, Name: "County", Depth: 1})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	if l1.Code != "L-1" || l2.Code != "L-2" {
		t.Fatalf("Codes = %q, %q, want L-1, L-2 (sequential across repeated Creates)", l1.Code, l2.Code)
	}

	renamedLevel, err := mgr.RenameLevel(ctx, l1.ID, "Ward Renamed")
	if err != nil {
		t.Fatalf("RenameLevel: %v", err)
	}
	if renamedLevel.Code != l1.Code {
		t.Fatalf("Code changed after level rename: got %q, want %q", renamedLevel.Code, l1.Code)
	}
}

func TestCreateLevelValidatesHierarchyAndAnchorExclusivity(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: "unknown", Name: "Ward"}); !errors.Is(err, mwanachamaactor.ErrHierarchyNotFound) {
		t.Fatalf("expected ErrHierarchyNotFound, got %v", err)
	}

	h, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Kenya"})

	ward, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Ward", Depth: 0, IsDefaultAnchor: true})
	if err != nil {
		t.Fatalf("CreateLevel(ward): %v", err)
	}
	if ward.ID == "" {
		t.Fatal("expected minted level id")
	}

	if _, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "County", Depth: 1, IsDefaultAnchor: true}); !errors.Is(err, mwanachamaactor.ErrDuplicateDefaultAnchor) {
		t.Fatalf("expected ErrDuplicateDefaultAnchor, got %v", err)
	}

	county, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "County", Depth: 1})
	if err != nil {
		t.Fatalf("CreateLevel(county, not anchor): %v", err)
	}

	got, err := mgr.GetLevel(ctx, county.ID)
	if err != nil {
		t.Fatalf("GetLevel: %v", err)
	}
	if got.Name != "County" || got.Depth != 1 {
		t.Fatalf("wrong level: %+v", got)
	}
	if _, err := mgr.GetLevel(ctx, "unknown"); !errors.Is(err, mwanachamaactor.ErrLevelNotFound) {
		t.Fatalf("expected ErrLevelNotFound, got %v", err)
	}

	renamed, err := mgr.RenameLevel(ctx, county.ID, "County v2")
	if err != nil {
		t.Fatalf("RenameLevel: %v", err)
	}
	if renamed.Name != "County v2" || renamed.Depth != 1 {
		t.Fatalf("rename changed more than the name: %+v", renamed)
	}

	levels, err := mgr.ListLevels(ctx, h.ID)
	if err != nil {
		t.Fatalf("ListLevels: %v", err)
	}
	if len(levels) != 2 || levels[0].ID != ward.ID || levels[1].ID != county.ID {
		t.Fatalf("expected depth order [ward, county], got %+v", levels)
	}
}

func TestSetDefaultAnchor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Kenya"})
	ward, _ := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Ward", Depth: 0, IsDefaultAnchor: true})
	county, _ := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "County", Depth: 1})

	if err := mgr.SetDefaultAnchor(ctx, h.ID, county.ID); err != nil {
		t.Fatalf("SetDefaultAnchor: %v", err)
	}
	levels, _ := mgr.ListLevels(ctx, h.ID)
	for _, l := range levels {
		want := l.ID == county.ID
		if l.IsDefaultAnchor != want {
			t.Fatalf("expected only county to carry the anchor flag, got %+v", levels)
		}
	}

	// No-op: re-setting the rung that already carries the flag.
	if err := mgr.SetDefaultAnchor(ctx, h.ID, county.ID); err != nil {
		t.Fatalf("SetDefaultAnchor (no-op): %v", err)
	}

	// A level from a different hierarchy is refused.
	other, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Other"})
	if err := mgr.SetDefaultAnchor(ctx, other.ID, ward.ID); !errors.Is(err, mwanachamaactor.ErrLevelNotFound) {
		t.Fatalf("expected ErrLevelNotFound for cross-hierarchy level, got %v", err)
	}
}

func TestDeleteLevelRefusedWhileWornOrScoped(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Kenya"})
	ward, _ := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Ward", Depth: 0})
	county, _ := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "County", Depth: 1})
	empty, _ := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Empty", Depth: 2})

	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "root", HierarchyID: h.ID, LevelID: ward.ID}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := mgr.DeleteLevel(ctx, ward.ID); !errors.Is(err, mwanachamaactor.ErrLevelWornByGroups) {
		t.Fatalf("expected ErrLevelWornByGroups, got %v", err)
	}

	k, _ := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator", LevelID: county.ID})
	if err := mgr.DeleteLevel(ctx, county.ID); !errors.Is(err, mwanachamaactor.ErrLevelHasRoleKinds) {
		t.Fatalf("expected ErrLevelHasRoleKinds, got %v", err)
	}
	_ = k

	if err := mgr.DeleteLevel(ctx, empty.ID); err != nil {
		t.Fatalf("DeleteLevel(empty): %v", err)
	}
	if _, err := mgr.GetLevel(ctx, empty.ID); !errors.Is(err, mwanachamaactor.ErrLevelNotFound) {
		t.Fatalf("expected level gone after delete, got %v", err)
	}

	if err := mgr.DeleteLevel(ctx, "unknown"); !errors.Is(err, mwanachamaactor.ErrLevelNotFound) {
		t.Fatalf("expected ErrLevelNotFound deleting an unknown level, got %v", err)
	}
}

func TestHierarchyCheckerMethods(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Kenya"})
	lvl, _ := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Ward", Depth: 0})
	other, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Other"})

	if ok, err := mgr.HierarchyExists(ctx, h.ID); err != nil || !ok {
		t.Fatalf("HierarchyExists(real) = %v, %v", ok, err)
	}
	if ok, err := mgr.HierarchyExists(ctx, "unknown"); err != nil || ok {
		t.Fatalf("HierarchyExists(unknown) = %v, %v", ok, err)
	}

	if ok, err := mgr.LevelInHierarchy(ctx, lvl.ID, h.ID); err != nil || !ok {
		t.Fatalf("LevelInHierarchy(same hierarchy) = %v, %v", ok, err)
	}
	if ok, err := mgr.LevelInHierarchy(ctx, lvl.ID, other.ID); err != nil || ok {
		t.Fatalf("LevelInHierarchy(other hierarchy) = %v, %v", ok, err)
	}
	if ok, err := mgr.LevelInHierarchy(ctx, "unknown", h.ID); err != nil || ok {
		t.Fatalf("LevelInHierarchy(unknown level) = %v, %v", ok, err)
	}
}
