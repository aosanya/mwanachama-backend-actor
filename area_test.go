package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/geo"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func square(lon, lat, size float64) geo.Geometry {
	return geo.NewPolygon(geo.Ring{
		{lon, lat}, {lon + size, lat}, {lon + size, lat + size}, {lon, lat + size}, {lon, lat},
	})
}

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

func (f areaFixture) group(t *testing.T, level mwanachamaactor.Level, name string, outline geo.Geometry) mwanachamaactor.Group {
	t.Helper()
	g, err := f.mgr.CreateGroup(context.Background(), mwanachamaactor.Group{
		HierarchyID: f.hierarchy.ID, LevelID: level.ID, Name: name, Outline: outline,
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

func TestAnOutlineRoundTripsOnAGroupAtAnAreaLevel(t *testing.T) {
	f := newAreaFixture(t)
	g := f.group(t, f.area, "North", square(10, 10, 1))
	got, err := f.mgr.GetGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.Outline.Kind != "Polygon" || len(got.Outline.Polygon[0]) != 5 {
		t.Fatalf("outline = %+v, want the stored square", got.Outline)
	}
}

func TestAGroupOffAnAreaLevelRefusesAnOutline(t *testing.T) {
	f := newAreaFixture(t)
	_, err := f.mgr.CreateGroup(context.Background(), mwanachamaactor.Group{
		HierarchyID: f.hierarchy.ID, LevelID: f.top.ID, Name: "Root", Outline: square(0, 0, 1),
	})
	if !errors.Is(err, mwanachamaactor.ErrNotAnArea) {
		t.Fatalf("err = %v, want ErrNotAnArea", err)
	}
	root := f.group(t, f.top, "Root", geo.Geometry{})
	if _, err := f.mgr.SetGroupOutline(context.Background(), root.ID, square(0, 0, 1)); !errors.Is(err, mwanachamaactor.ErrNotAnArea) {
		t.Fatalf("SetGroupOutline err = %v, want ErrNotAnArea", err)
	}
}

func TestAnOutlineIsAPolygonNeverAPoint(t *testing.T) {
	f := newAreaFixture(t)
	g := f.group(t, f.area, "North", geo.Geometry{})
	if _, err := f.mgr.SetGroupOutline(context.Background(), g.ID, geo.NewPoint(1, 1)); !errors.Is(err, mwanachamaactor.ErrInvalidOutline) {
		t.Fatalf("err = %v, want ErrInvalidOutline", err)
	}
}

func TestResolvePointAnswersTheAreaAPinFallsIn(t *testing.T) {
	f := newAreaFixture(t)
	ctx := context.Background()
	f.group(t, f.area, "North", square(10, 10, 1))
	south := f.group(t, f.area, "South", geo.Geometry{})
	if _, err := f.mgr.SetGroupOutline(ctx, south.ID, square(10, 8, 1)); err != nil {
		t.Fatalf("SetGroupOutline: %v", err)
	}
	got, ok, err := f.mgr.ResolvePoint(ctx, f.hierarchy.ID, 10.5, 8.5)
	if err != nil || !ok || got.ID != south.ID {
		t.Fatalf("ResolvePoint = %+v, %v, %v; want South", got, ok, err)
	}
	if _, ok, err := f.mgr.ResolvePoint(ctx, f.hierarchy.ID, 50, 50); err != nil || ok {
		t.Fatalf("ResolvePoint(outside) = %v, %v; want none", ok, err)
	}
}

func TestAnAgencyWithNoAreaLevelResolvesNone(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	h, _ := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Flat"})
	if _, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Only"}); err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	if _, ok, err := mgr.ResolvePoint(ctx, h.ID, 0, 0); err != nil || ok {
		t.Fatalf("ResolvePoint = %v, %v; want none", ok, err)
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
	for role, column := range map[string]string{"group": "outline", "level": "is_area"} {
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
	g := f.group(t, f.area, "North", square(1, 1, 1))
	if got, ok, err := f.mgr.ResolvePoint(ctx, f.hierarchy.ID, 1.5, 1.5); err != nil || !ok || got.ID != g.ID {
		t.Fatalf("ResolvePoint on the restored columns = %+v, %v, %v", got, ok, err)
	}
}
