package mwanachamaactor_test

// Covers board row ACT2: renaming a GroupType carries its Groups' NodeType
// with it, so DeleteGroupType's worn-by-Groups check (which reads the current
// name) still sees them and refuses the delete.

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestDeleteGroupType_StillRefusedAfterRename(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "H"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	level, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "National", IsDefaultAnchor: true})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	dept, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "Department"})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	grp, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{
		HierarchyID: h.ID, LevelID: level.ID, Name: "Comms", NodeType: "Department",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	// Confirm the guard works BEFORE any rename — this is the behavior
	// TestGroupType_DeleteWornRefused already covers, kept here only as a
	// control so the contrast with the post-rename case is explicit.
	if err := mgr.DeleteGroupType(ctx, dept.ID); !errors.Is(err, mwanachamaactor.ErrGroupTypeWornByGroups) {
		t.Fatalf("expected ErrGroupTypeWornByGroups before any rename, got %v", err)
	}

	// Rename the GroupType.
	if _, err := mgr.EditGroupType(ctx, dept.ID, "Committee", "", "", ""); err != nil {
		t.Fatalf("EditGroupType: %v", err)
	}

	got, err := mgr.GetGroup(ctx, grp.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.NodeType != "Committee" {
		t.Fatalf("expected the Group's NodeType to follow the rename to %q, got %q", "Committee", got.NodeType)
	}

	if err := mgr.DeleteGroupType(ctx, dept.ID); !errors.Is(err, mwanachamaactor.ErrGroupTypeWornByGroups) {
		t.Fatalf("expected ErrGroupTypeWornByGroups after rename, got %v", err)
	}
	if _, err := mgr.GetGroupType(ctx, dept.ID); err != nil {
		t.Fatalf("the GroupType must still exist, got %v", err)
	}
}
