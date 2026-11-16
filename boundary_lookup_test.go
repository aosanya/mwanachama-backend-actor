package mwanachamaactor_test

import (
	"context"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func boundedGroup(t *testing.T, um actor.UserManager, name string, b *geo.Geometry) actor.Group {
	t.Helper()
	g, err := um.CreateGroup(context.Background(), actor.Group{Name: name, Boundary: b})
	if err != nil {
		t.Fatalf("CreateGroup %s: %v", name, err)
	}
	return g
}

func TestGroupsAtPointFindsEveryGroupHoldingIt(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	town := geo.NewPolygon(ring(0, 0, 10, 10))
	ward := geo.NewPolygon(ring(1, 1, 3, 3))
	other := geo.NewPolygon(ring(5, 5, 6, 6))
	a := boundedGroup(t, um, "Város", &town)
	b := boundedGroup(t, um, "Belső kerület", &ward)
	boundedGroup(t, um, "Külső kerület", &other)
	boundedGroup(t, um, "Határ nélkül", nil)

	got, err := um.GroupsAtPoint(ctx, 2, 2)
	if err != nil {
		t.Fatalf("GroupsAtPoint: %v", err)
	}
	ids := map[string]bool{}
	for _, g := range got {
		ids[g.ID] = true
	}
	if len(got) != 2 || !ids[a.ID] || !ids[b.ID] {
		t.Fatalf("groups at (2,2) = %v, want the town and the inner ward", got)
	}
	none, err := um.GroupsAtPoint(ctx, 50, 50)
	if err != nil || len(none) != 0 {
		t.Fatalf("groups at (50,50) = %v, %v, want none", none, err)
	}
}

func TestGroupsAtPointRefusesAPlaceOffTheEarth(t *testing.T) {
	um := newManager(t)
	if _, err := um.GroupsAtPoint(context.Background(), 200, 0); err == nil {
		t.Fatal("longitude 200 was accepted")
	}
	if _, err := um.GroupsAtPoint(context.Background(), 0, -91); err == nil {
		t.Fatal("latitude -91 was accepted")
	}
}

func TestGroupWardIndexIsBuiltFromBoundaries(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	sq := geo.NewPolygon(ring(0, 0, 2, 2))
	g := boundedGroup(t, um, "Keleti kerület", &sq)
	index, _, err := um.GroupWardIndex(ctx)
	if err != nil {
		t.Fatalf("GroupWardIndex: %v", err)
	}
	w, err := index.WardOf(geo.Position{1, 1})
	if err != nil || w.ID != g.ID || w.Name != "Keleti kerület" {
		t.Fatalf("ward = %v, %v", w, err)
	}
	if _, err := index.WardOf(geo.Position{9, 9}); err == nil {
		t.Fatal("a point outside every boundary found a ward")
	}
}
