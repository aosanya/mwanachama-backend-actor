package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func ring(minLon, minLat, maxLon, maxLat float64) geo.Ring {
	return geo.Ring{{minLon, minLat}, {maxLon, minLat}, {maxLon, maxLat}, {minLon, maxLat}, {minLon, minLat}}
}

func square() *geo.Geometry {
	g := geo.NewPolygon(ring(0, 0, 2, 2))
	return &g
}

func TestGroupBoundarySurvivesAWriteAndARead(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g, err := um.CreateGroup(ctx, actor.Group{Name: "Északi kerület", Boundary: square()})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	read, err := um.GetGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if read.Boundary == nil || !geo.Contains(*read.Boundary, geo.Position{1, 1}) || geo.Contains(*read.Boundary, geo.Position{5, 5}) {
		t.Fatalf("stored boundary = %v, want the square", read.Boundary)
	}
}

func TestSetGroupBoundarySetsAndClears(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g, err := um.CreateGroup(ctx, actor.Group{Name: "Déli kerület"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.Boundary != nil {
		t.Fatal("a new group has a boundary nobody gave it")
	}
	if _, err := um.SetGroupBoundary(ctx, g.ID, square()); err != nil {
		t.Fatalf("SetGroupBoundary: %v", err)
	}
	read, _ := um.GetGroup(ctx, g.ID)
	if read.Boundary == nil || read.Boundary.Kind != geo.KindPolygon {
		t.Fatalf("boundary after set = %v", read.Boundary)
	}
	if _, err := um.SetGroupBoundary(ctx, g.ID, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	read, _ = um.GetGroup(ctx, g.ID)
	if read.Boundary != nil {
		t.Fatalf("boundary after clear = %v", read.Boundary)
	}
}

func TestSetGroupBoundaryRefusesWhatIsNotAnArea(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g, _ := um.CreateGroup(ctx, actor.Group{Name: "Keleti kerület"})
	pt := geo.NewPoint(1, 1)
	if _, err := um.SetGroupBoundary(ctx, g.ID, &pt); !errors.Is(err, actor.ErrBoundaryNotAnArea) {
		t.Fatalf("point = %v, want ErrBoundaryNotAnArea", err)
	}
	open := geo.NewPolygon(geo.Ring{{0, 0}, {1, 0}, {1, 1}, {0, 1}})
	if _, err := um.SetGroupBoundary(ctx, g.ID, &open); err == nil {
		t.Fatal("an unclosed ring was accepted")
	}
	if _, err := um.CreateGroup(ctx, actor.Group{Name: "Nyugati kerület", Boundary: &pt}); !errors.Is(err, actor.ErrBoundaryNotAnArea) {
		t.Fatalf("CreateGroup with a point = %v", err)
	}
}
