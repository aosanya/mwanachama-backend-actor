package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func hierarchyNamed(t *testing.T, um actor.UserManager, name string) actor.Hierarchy {
	t.Helper()
	h, err := um.CreateHierarchy(context.Background(), actor.Hierarchy{Name: name})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	return h
}

func TestHierarchiesGetCodesRenameAndAreFoundByID(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	first := hierarchyNamed(t, um, "Közigazgatás")
	second := hierarchyNamed(t, um, "Egyház")
	if first.Code == "" || first.Code == second.Code {
		t.Fatalf("codes = %q and %q, want two different ones", first.Code, second.Code)
	}
	renamed, err := um.RenameHierarchy(ctx, first.ID, "Önkormányzat")
	if err != nil || renamed.Name != "Önkormányzat" || renamed.Code != first.Code {
		t.Fatalf("renamed = %+v, %v; the code must stay", renamed, err)
	}
	read, _ := um.GetHierarchy(ctx, first.ID)
	if read.Name != "Önkormányzat" {
		t.Fatalf("read after rename = %+v", read)
	}
	if _, err := um.GetHierarchy(ctx, "missing"); !errors.Is(err, actor.ErrHierarchyNotFound) {
		t.Errorf("unknown hierarchy: err = %v", err)
	}
	if _, err := um.RenameHierarchy(ctx, "missing", "x"); !errors.Is(err, actor.ErrHierarchyNotFound) {
		t.Errorf("rename unknown: err = %v", err)
	}
	all, err := um.ListHierarchies(ctx)
	if err != nil || len(all) != 2 {
		t.Fatalf("hierarchies = %+v, %v", all, err)
	}
	if ok, _ := um.HierarchyExists(ctx, first.ID); !ok {
		t.Error("HierarchyExists is false for a hierarchy that exists")
	}
	if ok, _ := um.HierarchyExists(ctx, ""); ok {
		t.Error("HierarchyExists is true for no id")
	}
	if ok, err := um.HierarchyExists(ctx, "missing"); ok || err != nil {
		t.Errorf("HierarchyExists(missing) = %v, %v", ok, err)
	}
}

func TestOnlyOneLevelOfAHierarchyIsTheDefaultAnchorAndItCanMove(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	h := hierarchyNamed(t, um, "Közigazgatás")
	other := hierarchyNamed(t, um, "Egyház")
	top, err := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Megye", Depth: 1, IsDefaultAnchor: true})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	if _, err := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Járás", Depth: 2, IsDefaultAnchor: true}); !errors.Is(err, actor.ErrDuplicateDefaultAnchor) {
		t.Fatalf("second anchor: err = %v, want ErrDuplicateDefaultAnchor", err)
	}
	if _, err := um.CreateLevel(ctx, actor.Level{HierarchyID: "missing", Name: "Város"}); !errors.Is(err, actor.ErrHierarchyNotFound) {
		t.Errorf("level in an unknown hierarchy: err = %v", err)
	}
	district, _ := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Járás", Depth: 2})
	foreign, _ := um.CreateLevel(ctx, actor.Level{HierarchyID: other.ID, Name: "Plébánia", Depth: 1})
	if err := um.SetDefaultAnchor(ctx, h.ID, district.ID); err != nil {
		t.Fatalf("SetDefaultAnchor: %v", err)
	}
	if err := um.SetDefaultAnchor(ctx, h.ID, district.ID); err != nil {
		t.Fatalf("SetDefaultAnchor again: %v", err)
	}
	moved, _ := um.GetLevel(ctx, district.ID)
	was, _ := um.GetLevel(ctx, top.ID)
	if !moved.IsDefaultAnchor || was.IsDefaultAnchor {
		t.Fatalf("anchor after the move: district=%v top=%v", moved.IsDefaultAnchor, was.IsDefaultAnchor)
	}
	if err := um.SetDefaultAnchor(ctx, h.ID, foreign.ID); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("a level of another hierarchy: err = %v", err)
	}
	if err := um.SetDefaultAnchor(ctx, h.ID, "missing"); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("unknown level: err = %v", err)
	}
}

func TestLevelsListByDepthRenameAndBelongToTheirHierarchy(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	h := hierarchyNamed(t, um, "Közigazgatás")
	other := hierarchyNamed(t, um, "Egyház")
	deep, _ := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Város", Depth: 3})
	top, _ := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Megye", Depth: 1})
	um.CreateLevel(ctx, actor.Level{HierarchyID: other.ID, Name: "Plébánia", Depth: 1})
	got, err := um.ListLevels(ctx, h.ID)
	if err != nil || len(got) != 2 || got[0].ID != top.ID || got[1].ID != deep.ID {
		t.Fatalf("levels = %+v, %v; want shallowest first and only this hierarchy's", got, err)
	}
	every, _ := um.ListLevels(ctx, "")
	if len(every) != 3 {
		t.Fatalf("all levels = %d, want 3", len(every))
	}
	renamed, err := um.RenameLevel(ctx, top.ID, "Vármegye")
	if err != nil || renamed.Name != "Vármegye" {
		t.Fatalf("renamed = %+v, %v", renamed, err)
	}
	if _, err := um.RenameLevel(ctx, "missing", "x"); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("rename unknown: err = %v", err)
	}
	if ok, _ := um.LevelInHierarchy(ctx, top.ID, h.ID); !ok {
		t.Error("LevelInHierarchy is false for its own hierarchy")
	}
	if ok, _ := um.LevelInHierarchy(ctx, top.ID, other.ID); ok {
		t.Error("LevelInHierarchy is true for another hierarchy")
	}
}

func TestALevelWornByAGroupCannotBeRemovedButAFreeOneCan(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	h := hierarchyNamed(t, um, "Közigazgatás")
	worn, _ := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Megye", Depth: 1})
	free, _ := um.CreateLevel(ctx, actor.Level{HierarchyID: h.ID, Name: "Város", Depth: 2})
	if _, err := um.CreateGroup(ctx, actor.Group{Name: "Pest", HierarchyID: h.ID, LevelID: worn.ID}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := um.DeleteLevel(ctx, worn.ID); !errors.Is(err, actor.ErrLevelWornByGroups) {
		t.Fatalf("worn level: err = %v, want ErrLevelWornByGroups", err)
	}
	if err := um.DeleteLevel(ctx, free.ID); err != nil {
		t.Fatalf("free level: %v", err)
	}
	if _, err := um.GetLevel(ctx, free.ID); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("deleted level still found: %v", err)
	}
	if err := um.DeleteLevel(ctx, "missing"); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("delete unknown: err = %v", err)
	}
}

func TestGroupTypesGetDefaultsAreEditedAndRenameTheGroupsThatWearThem(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	h := hierarchyNamed(t, um, "Egyház")
	if _, err := um.CreateGroupType(ctx, actor.GroupType{HierarchyID: h.ID, Name: " "}); !errors.Is(err, actor.ErrInvalidGroupType) {
		t.Errorf("blank name: err = %v", err)
	}
	if _, err := um.CreateGroupType(ctx, actor.GroupType{HierarchyID: "missing", Name: "Parish"}); !errors.Is(err, actor.ErrHierarchyNotFound) {
		t.Errorf("unknown hierarchy: err = %v", err)
	}
	if _, err := um.CreateGroupType(ctx, actor.GroupType{HierarchyID: h.ID, Name: "Parish", LevelID: "missing"}); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("unknown level: err = %v", err)
	}
	gt, err := um.CreateGroupType(ctx, actor.GroupType{HierarchyID: h.ID, Name: "Parish"})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	if gt.Singular != "Parish" || gt.Plural != "Parishs" || gt.Code == "" {
		t.Fatalf("group type = %+v, want the name as singular and name+s as plural", gt)
	}
	member, err := um.CreateGroup(ctx, actor.Group{Name: "Szent István", HierarchyID: h.ID, NodeType: "Parish"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := um.EditGroupType(ctx, gt.ID, " ", "", "", ""); !errors.Is(err, actor.ErrInvalidGroupType) {
		t.Errorf("edit to a blank name: err = %v", err)
	}
	if _, err := um.EditGroupType(ctx, gt.ID, "Church", "", "", "missing"); !errors.Is(err, actor.ErrLevelNotFound) {
		t.Errorf("edit to an unknown level: err = %v", err)
	}
	edited, err := um.EditGroupType(ctx, gt.ID, "Church", "A church", "Churches", "")
	if err != nil || edited.Name != "Church" || edited.Singular != "A church" || edited.Plural != "Churches" {
		t.Fatalf("edited = %+v, %v", edited, err)
	}
	g, _ := um.GetGroup(ctx, member.ID)
	if g.NodeType != "Church" {
		t.Fatalf("the group's type = %q, want it renamed with its type", g.NodeType)
	}
	if err := um.DeleteGroupType(ctx, gt.ID); !errors.Is(err, actor.ErrGroupTypeWornByGroups) {
		t.Fatalf("delete a worn type: err = %v", err)
	}
	if _, err := um.EditGroupType(ctx, "missing", "x", "", "", ""); !errors.Is(err, actor.ErrGroupTypeNotFound) {
		t.Errorf("edit unknown: err = %v", err)
	}
}

func TestAFreeGroupTypeCanBeDeletedAndListsByCodeForOneHierarchy(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	h := hierarchyNamed(t, um, "Egyház")
	other := hierarchyNamed(t, um, "Közigazgatás")
	a, _ := um.CreateGroupType(ctx, actor.GroupType{HierarchyID: h.ID, Name: "Parish"})
	b, _ := um.CreateGroupType(ctx, actor.GroupType{HierarchyID: h.ID, Name: "Diocese"})
	um.CreateGroupType(ctx, actor.GroupType{HierarchyID: other.ID, Name: "Town"})
	got, err := um.ListGroupTypes(ctx, h.ID)
	if err != nil || len(got) != 2 || got[0].ID != a.ID || got[1].ID != b.ID {
		t.Fatalf("types = %+v, %v; want this hierarchy's two in code order", got, err)
	}
	if err := um.DeleteGroupType(ctx, a.ID); err != nil {
		t.Fatalf("DeleteGroupType: %v", err)
	}
	if _, err := um.GetGroupType(ctx, a.ID); !errors.Is(err, actor.ErrGroupTypeNotFound) {
		t.Errorf("deleted type still found: %v", err)
	}
	if err := um.DeleteGroupType(ctx, "missing"); !errors.Is(err, actor.ErrGroupTypeNotFound) {
		t.Errorf("delete unknown: err = %v", err)
	}
}

func TestCreateGroupRefusesNoNameAndABadJoinModeAndDefaultsToConfirmed(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	if _, err := um.CreateGroup(ctx, actor.Group{}); !errors.Is(err, actor.ErrInvalidGroup) {
		t.Errorf("no name: err = %v", err)
	}
	if _, err := um.CreateGroup(ctx, actor.Group{Name: "Pest", JoinMode: "by invitation"}); !errors.Is(err, actor.ErrInvalidJoinMode) {
		t.Errorf("bad join mode: err = %v", err)
	}
	g, err := um.CreateGroup(ctx, actor.Group{Name: "Pest"})
	if err != nil || g.JoinMode != models.JoinConfirmed || g.Code == "" || g.PublicCode == "" {
		t.Fatalf("group = %+v, %v", g, err)
	}
	open, _ := um.CreateGroup(ctx, actor.Group{Name: "Buda", JoinMode: models.JoinOpen})
	if !open.Open() {
		t.Fatal("an open group is not open")
	}
}

func TestEditingAGroupChangesNameTypeAndAnchorAndStampsTheTime(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g, _ := um.CreateGroup(ctx, actor.Group{Name: "Pest"})
	edited, err := um.EditGroup(ctx, g.ID, actor.GroupEdit{Name: "Pest megye", NodeType: "County"})
	if err != nil || edited.Name != "Pest megye" || edited.NodeType != "County" {
		t.Fatalf("edited = %+v, %v", edited, err)
	}
	read, _ := um.GetGroup(ctx, g.ID)
	if read.Name != "Pest megye" || read.NodeType != "County" || read.LastUpdated < g.LastUpdated {
		t.Fatalf("read = %+v", read)
	}
	if _, err := um.EditGroup(ctx, "missing", actor.GroupEdit{Name: "x"}); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("edit unknown: err = %v", err)
	}
}

func TestMovingAGroupRefusesItselfTheRootItsOwnSubtreeAndAMissingParent(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	root := groupNamed(t, um, "Magyarország", "")
	county := groupNamed(t, um, "Pest", root.ID)
	town := groupNamed(t, um, "Vác", county.ID)
	other := groupNamed(t, um, "Nógrád", root.ID)
	if _, err := um.MoveGroup(ctx, county.ID, county.ID); !errors.Is(err, actor.ErrParentIsSelf) {
		t.Errorf("under itself: err = %v", err)
	}
	if _, err := um.MoveGroup(ctx, root.ID, other.ID); !errors.Is(err, actor.ErrRootCannotMove) {
		t.Errorf("the root: err = %v", err)
	}
	if _, err := um.MoveGroup(ctx, county.ID, town.ID); !errors.Is(err, actor.ErrParentInSubtree) {
		t.Errorf("into its own subtree: err = %v", err)
	}
	if _, err := um.MoveGroup(ctx, county.ID, "missing"); !errors.Is(err, actor.ErrParentNotFound) {
		t.Errorf("missing parent: err = %v", err)
	}
	if _, err := um.MoveGroup(ctx, "missing", other.ID); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("missing group: err = %v", err)
	}
	moved, err := um.MoveGroup(ctx, town.ID, other.ID)
	if err != nil || moved.ParentID != other.ID {
		t.Fatalf("moved = %+v, %v", moved, err)
	}
	kids, _ := um.ListGroupChildren(ctx, other.ID)
	if len(kids) != 1 || kids[0].ID != town.ID {
		t.Fatalf("children of the new parent = %+v", kids)
	}
	if left, _ := um.ListGroupChildren(ctx, county.ID); len(left) != 0 {
		t.Fatalf("children left under the old parent = %+v", left)
	}
}

func TestDiscoverableGroupsListByNameCaseInsensitivelyAndSkipHiddenOnes(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	um.CreateGroup(ctx, actor.Group{Name: "Szent István plébánia", Discoverable: true})
	um.CreateGroup(ctx, actor.Group{Name: "Szent László plébánia", Discoverable: true})
	um.CreateGroup(ctx, actor.Group{Name: "Szent Imre kör"})
	all, err := um.ListDiscoverableGroups(ctx, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("discoverable = %+v, %v; want the two marked discoverable", all, err)
	}
	found, _ := um.ListDiscoverableGroups(ctx, "ISTVÁN")
	if len(found) != 1 || found[0].Name != "Szent István plébánia" {
		t.Fatalf("search = %+v", found)
	}
	if none, _ := um.ListDiscoverableGroups(ctx, "Imre"); len(none) != 0 {
		t.Fatalf("a group not marked discoverable was listed: %+v", none)
	}
}
