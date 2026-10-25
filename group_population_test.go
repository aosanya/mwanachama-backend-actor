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

func TestAWithdrawnFigureLeavesTheLatestReadButStaysInTheHistory(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 3"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	only, err := mgr.RecordGroupPopulation(ctx, g.ID, 500, "2022-08-01", "register", "op-1")
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	w, err := mgr.WithdrawGroupPopulation(ctx, g.ID, only.ID, "op-2")
	if err != nil || w.Withdraws != only.ID || w.RecordedBy != "op-2" {
		t.Fatalf("withdraw = %+v, %v", w, err)
	}
	if _, err := mgr.LatestGroupPopulation(ctx, g.ID); !errors.Is(err, mwanachamaactor.ErrPopulationNotFound) {
		t.Fatalf("latest after withdrawing the only figure: err = %v, want ErrPopulationNotFound", err)
	}
	all, err := mgr.ListGroupPopulations(ctx, g.ID)
	if err != nil || len(all) != 2 {
		t.Fatalf("history = %+v, %v; want the figure and its withdrawn record", all, err)
	}
	if _, err := mgr.WithdrawGroupPopulation(ctx, g.ID, w.ID, "op-2"); !errors.Is(err, mwanachamaactor.ErrPopulationNotFound) {
		t.Errorf("withdrawing a withdrawn record: err = %v, want ErrPopulationNotFound", err)
	}
	other, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.WithdrawGroupPopulation(ctx, other.ID, only.ID, "op-2"); !errors.Is(err, mwanachamaactor.ErrPopulationNotFound) {
		t.Errorf("withdrawing another group's figure: err = %v, want ErrPopulationNotFound", err)
	}
}

func TestAParentWithNoFigureReadsTheSumOfItsInnerGroupsLatestFigures(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	city, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "City"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	district, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "District", ParentID: city.ID})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	wardA, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward A", ParentID: district.ID})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	wardB, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward B", ParentID: district.ID})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	east, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "East", ParentID: city.ID})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.LatestGroupPopulation(ctx, city.ID); !errors.Is(err, mwanachamaactor.ErrPopulationNotFound) {
		t.Fatalf("no figures anywhere: err = %v, want ErrPopulationNotFound", err)
	}
	mustRecord := func(groupID string, n int, asOf string) {
		t.Helper()
		if _, err := mgr.RecordGroupPopulation(ctx, groupID, n, asOf, "census", "op-1"); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	mustRecord(wardA.ID, 100, "2019-08-01")
	mustRecord(wardA.ID, 300, "2022-08-01")
	mustRecord(wardB.ID, 200, "2021-08-01")
	mustRecord(east.ID, 50, "2022-08-01")

	got, err := mgr.LatestGroupPopulation(ctx, city.ID)
	if err != nil || got.Population != 550 || got.GroupID != city.ID || got.AsOf != "2021-08-01" {
		t.Fatalf("city = %+v, %v; want 300+200+50 as of the oldest part", got, err)
	}

	own, err := mgr.RecordGroupPopulation(ctx, district.ID, 1000, "2020-08-01", "planner", "op-1")
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got, err := mgr.LatestGroupPopulation(ctx, city.ID); err != nil || got.Population != 1050 {
		t.Fatalf("city with a district figure = %+v, %v; want 1000+50, the district's own figure replacing its wards' sum", got, err)
	}
	if _, err := mgr.WithdrawGroupPopulation(ctx, district.ID, own.ID, "op-1"); err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if got, err := mgr.LatestGroupPopulation(ctx, district.ID); err != nil || got.Population != 500 {
		t.Fatalf("district after withdrawing its own figure = %+v, %v; want its wards' sum", got, err)
	}
}
