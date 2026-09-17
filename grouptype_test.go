package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestGroupTypeCRUD(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Umoja National Party Structure"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	level, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Ward", Depth: 3})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	dept, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "Department"})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	if dept.Code != "GT-1" {
		t.Fatalf("expected first GroupType to mint GT-1, got %q", dept.Code)
	}
	if dept.Singular != "Department" || dept.Plural != "Departments" {
		t.Fatalf("expected label fallback, got singular=%q plural=%q", dept.Singular, dept.Plural)
	}

	ward, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "Ward Branch", LevelID: level.ID})
	if err != nil {
		t.Fatalf("CreateGroupType(Ward Branch): %v", err)
	}

	all, err := mgr.ListGroupTypes(ctx, h.ID)
	if err != nil {
		t.Fatalf("ListGroupTypes: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 group types, got %d", len(all))
	}

	renamed, err := mgr.EditGroupType(ctx, ward.ID, "Branch", "", "", level.ID)
	if err != nil {
		t.Fatalf("EditGroupType: %v", err)
	}
	if renamed.Name != "Branch" || renamed.Plural != "Branchs" {
		t.Fatalf("expected labels to follow the new name, got %+v", renamed)
	}
	if renamed.Code != ward.Code {
		t.Fatalf("expected Code to survive an edit, got %q (was %q)", renamed.Code, ward.Code)
	}

	if err := mgr.DeleteGroupType(ctx, ward.ID); err != nil {
		t.Fatalf("DeleteGroupType: %v", err)
	}
	if _, err := mgr.GetGroupType(ctx, ward.ID); !errors.Is(err, mwanachamaactor.ErrGroupTypeNotFound) {
		t.Fatalf("expected ErrGroupTypeNotFound, got %v", err)
	}
}

func TestGroupType_NameRequired(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "H"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}

	if _, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "   "}); !errors.Is(err, mwanachamaactor.ErrInvalidGroupType) {
		t.Fatalf("expected ErrInvalidGroupType, got %v", err)
	}

	gt, err := mgr.CreateGroupType(ctx, mwanachamaactor.GroupType{HierarchyID: h.ID, Name: "Committee"})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	if _, err := mgr.EditGroupType(ctx, gt.ID, "", "", "", ""); !errors.Is(err, mwanachamaactor.ErrInvalidGroupType) {
		t.Fatalf("expected ErrInvalidGroupType on edit, got %v", err)
	}
}

// TestGroupType_DeleteWornRefused mirrors DeleteLevel's own worn-by-Groups
// guard: NodeType is plain free text (see Group.NodeType's doc), so the
// check is by name, matched against whatever a caller — including one that
// never heard of GroupType — wrote there.
func TestGroupType_DeleteWornRefused(t *testing.T) {
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
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{
		HierarchyID: h.ID, LevelID: level.ID, Name: "Communications & Media", NodeType: "Department",
	}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	if err := mgr.DeleteGroupType(ctx, dept.ID); !errors.Is(err, mwanachamaactor.ErrGroupTypeWornByGroups) {
		t.Fatalf("expected ErrGroupTypeWornByGroups, got %v", err)
	}
}
