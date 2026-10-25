package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

type areaFixture struct {
	mgr       mwanachamaactor.UserManager
	hierarchy mwanachamaactor.Hierarchy
	top       mwanachamaactor.Level
	area      mwanachamaactor.Level
}

func newAreaFixture(t *testing.T) areaFixture {
	t.Helper()
	ctx := context.Background()
	mgr := newTestManager(t)
	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Tree"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	top, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Top", Depth: 0})
	if err != nil {
		t.Fatalf("CreateLevel(top): %v", err)
	}
	area, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "District", Depth: 1, IsArea: true})
	if err != nil {
		t.Fatalf("CreateLevel(area): %v", err)
	}
	return areaFixture{mgr: mgr, hierarchy: h, top: top, area: area}
}

func (f areaFixture) group(t *testing.T, level mwanachamaactor.Level, name string, areaNodeID string) mwanachamaactor.Group {
	t.Helper()
	g, err := f.mgr.CreateGroup(context.Background(), mwanachamaactor.Group{
		HierarchyID: f.hierarchy.ID, LevelID: level.ID, Name: name, AreaNodeID: areaNodeID,
	})
	if err != nil {
		t.Fatalf("CreateGroup(%s): %v", name, err)
	}
	return g
}

func TestALevelMarkedAnAreaKeepsTheTemplatesOwnName(t *testing.T) {
	f := newAreaFixture(t)
	got, err := f.mgr.GetLevel(context.Background(), f.area.ID)
	if err != nil {
		t.Fatalf("GetLevel: %v", err)
	}
	if !got.IsArea || got.Name != "District" {
		t.Fatalf("level = %+v, want an area named District", got)
	}
}

func TestAnAreaNodeRoundTripsOnAGroupAtAnAreaLevel(t *testing.T) {
	f := newAreaFixture(t)
	g := f.group(t, f.area, "North", "node-north")
	got, err := f.mgr.GetGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.AreaNodeID != "node-north" {
		t.Fatalf("area node = %q, want node-north", got.AreaNodeID)
	}
}

func TestAGroupOffAnAreaLevelRefusesAnAreaNode(t *testing.T) {
	f := newAreaFixture(t)
	_, err := f.mgr.CreateGroup(context.Background(), mwanachamaactor.Group{
		HierarchyID: f.hierarchy.ID, LevelID: f.top.ID, Name: "Root", AreaNodeID: "node-root",
	})
	if !errors.Is(err, mwanachamaactor.ErrNotAnArea) {
		t.Fatalf("err = %v, want ErrNotAnArea", err)
	}
	root := f.group(t, f.top, "Root", "")
	if _, err := f.mgr.SetGroupAreaNode(context.Background(), root.ID, "node-root"); !errors.Is(err, mwanachamaactor.ErrNotAnArea) {
		t.Fatalf("SetGroupAreaNode err = %v, want ErrNotAnArea", err)
	}
}

func TestSettingAnAreaNodeReplacesAndClearsIt(t *testing.T) {
	f := newAreaFixture(t)
	ctx := context.Background()
	g := f.group(t, f.area, "South", "")
	if _, err := f.mgr.SetGroupAreaNode(ctx, g.ID, "node-south"); err != nil {
		t.Fatalf("SetGroupAreaNode: %v", err)
	}
	if got, _ := f.mgr.GetGroup(ctx, g.ID); got.AreaNodeID != "node-south" {
		t.Fatalf("after set = %q, want node-south", got.AreaNodeID)
	}
	if _, err := f.mgr.SetGroupAreaNode(ctx, g.ID, ""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got, _ := f.mgr.GetGroup(ctx, g.ID); got.AreaNodeID != "" {
		t.Fatalf("after clear = %q, want empty", got.AreaNodeID)
	}
}

func TestProvisionAddsTheAreaColumnsOnToExistingTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	newTestManagerOn(t, db)
	for role, column := range map[string]string{"group": "area_node_id", "level": "is_area"} {
		o, _ := s.ByRole(role)
		if err := db.Exec("alter table " + s.TableFor(o) + " drop column " + column).Error; err != nil {
			t.Fatalf("drop %s: %v", column, err)
		}
	}
	f := areaFixture{mgr: newTestManagerOn(t, db)}
	ctx := context.Background()
	f.hierarchy, _ = f.mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Tree"})
	f.area, err = f.mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: f.hierarchy.ID, Name: "Ward", IsArea: true})
	if err != nil {
		t.Fatalf("CreateLevel on the restored column: %v", err)
	}
	g := f.group(t, f.area, "North", "node-north")
	if got, err := f.mgr.GetGroup(ctx, g.ID); err != nil || got.AreaNodeID != "node-north" {
		t.Fatalf("area node on the restored column = %+v, %v", got, err)
	}
}
