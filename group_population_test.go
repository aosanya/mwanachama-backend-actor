package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestAGroupsLatestPopulationIsTheLatestDatedFigureAndEveryFigureIsKept(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.LatestGroupPopulation(ctx, g.ID); !errors.Is(err, mwanachamaactor.ErrPopulationNotFound) {
		t.Fatalf("latest before any figure: err = %v, want ErrPopulationNotFound", err)
	}
	if _, err := mgr.RecordGroupPopulation(ctx, g.ID, 12000, "2022-08-01", "2022 census", "op-1"); err != nil {
		t.Fatalf("RecordGroupPopulation: %v", err)
	}
	if _, err := mgr.RecordGroupPopulation(ctx, g.ID, 9000, "2019-08-01", "2019 census", "op-1"); err != nil {
		t.Fatalf("RecordGroupPopulation older: %v", err)
	}
	latest, err := mgr.LatestGroupPopulation(ctx, g.ID)
	if err != nil || latest.Population != 12000 || latest.Source != "2022 census" || latest.RecordedBy != "op-1" {
		t.Fatalf("latest = %+v, %v; want the 2022 figure", latest, err)
	}
	all, err := mgr.ListGroupPopulations(ctx, g.ID)
	if err != nil || len(all) != 2 || all[0].AsOf != "2019-08-01" || all[1].AsOf != "2022-08-01" {
		t.Fatalf("history = %+v, %v; want both, oldest first", all, err)
	}
}

func TestAPopulationFigureNeedsACountADateAndASource(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 2"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	cases := []struct {
		name         string
		population   int
		asOf, source string
	}{
		{"negative", -1, "2022-08-01", "census"},
		{"undated", 10, "", "census"},
		{"not a date", 10, "August 2022", "census"},
		{"unsourced", 10, "2022-08-01", "  "},
	}
	for _, c := range cases {
		if _, err := mgr.RecordGroupPopulation(ctx, g.ID, c.population, c.asOf, c.source, "op-1"); !errors.Is(err, mwanachamaactor.ErrInvalidPopulation) {
			t.Errorf("%s: err = %v, want ErrInvalidPopulation", c.name, err)
		}
	}
	if _, err := mgr.RecordGroupPopulation(ctx, "no-such-group", 10, "2022-08-01", "census", "op-1"); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Errorf("unknown group: err = %v, want ErrGroupNotFound", err)
	}
}
