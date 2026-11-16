package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
)

func groupNamed(t *testing.T, um actor.UserManager, name, parent string) actor.Group {
	t.Helper()
	g, err := um.CreateGroup(context.Background(), actor.Group{Name: name, ParentID: parent})
	if err != nil {
		t.Fatalf("CreateGroup %s: %v", name, err)
	}
	return g
}

func TestPopulationRefusesANegativeFigureABadDateOrNoSource(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	cases := map[string]struct {
		population   int
		asOf, source string
	}{
		"negative":   {-1, "2026-01-01", "census"},
		"bad date":   {10, "1 Jan 2026", "census"},
		"no source":  {10, "2026-01-01", "  "},
		"no date":    {10, "", "census"},
		"impossible": {10, "2026-02-30", "census"},
	}
	for name, c := range cases {
		if _, err := um.RecordGroupPopulation(ctx, g.ID, c.population, c.asOf, c.source, "owner"); !errors.Is(err, actor.ErrInvalidPopulation) {
			t.Errorf("%s: err = %v, want ErrInvalidPopulation", name, err)
		}
	}
}

func TestPopulationOfAnUnknownGroupIsRefused(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	if _, err := um.RecordGroupPopulation(ctx, "nowhere", 5, "2026-01-01", "census", "owner"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("record: err = %v", err)
	}
	if _, err := um.ListGroupPopulations(ctx, "nowhere"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("list: err = %v", err)
	}
	if _, err := um.LatestGroupPopulation(ctx, "nowhere"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("latest: err = %v", err)
	}
}

func TestPopulationFiguresListByDateOldestFirst(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Déli kerület", "")
	um.RecordGroupPopulation(ctx, g.ID, 300, "2026-06-01", "estimate", "owner")
	um.RecordGroupPopulation(ctx, g.ID, 200, "2025-06-01", "census", "owner")
	got, err := um.ListGroupPopulations(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListGroupPopulations: %v", err)
	}
	if len(got) != 2 || got[0].Population != 200 || got[1].Population != 300 {
		t.Fatalf("figures = %+v, want 200 then 300", got)
	}
}

func TestLatestFigureIsTheNewestOneStanding(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Keleti kerület", "")
	um.RecordGroupPopulation(ctx, g.ID, 200, "2025-06-01", "census", "owner")
	newer, _ := um.RecordGroupPopulation(ctx, g.ID, 300, "2026-06-01", "estimate", "owner")
	latest, err := um.LatestGroupPopulation(ctx, g.ID)
	if err != nil || latest.Population != 300 {
		t.Fatalf("latest = %+v, %v; want 300", latest, err)
	}
	if _, err := um.WithdrawGroupPopulation(ctx, g.ID, newer.ID, "owner"); err != nil {
		t.Fatalf("WithdrawGroupPopulation: %v", err)
	}
	latest, err = um.LatestGroupPopulation(ctx, g.ID)
	if err != nil || latest.Population != 200 {
		t.Fatalf("latest after the withdraw = %+v, %v; want 200 again", latest, err)
	}
}

func TestWithdrawingTheSameFigureTwiceOrAnUnknownOneIsRefused(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Nyugati kerület", "")
	p, _ := um.RecordGroupPopulation(ctx, g.ID, 100, "2026-01-01", "census", "owner")
	w, err := um.WithdrawGroupPopulation(ctx, g.ID, p.ID, "owner")
	if err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if w.Withdraws != p.ID || w.Population != 100 || w.Source != "withdrawn" {
		t.Fatalf("withdrawal = %+v", w)
	}
	if _, err := um.WithdrawGroupPopulation(ctx, g.ID, p.ID, "owner"); !errors.Is(err, actor.ErrPopulationNotFound) {
		t.Errorf("second withdraw: err = %v", err)
	}
	if _, err := um.WithdrawGroupPopulation(ctx, g.ID, w.ID, "owner"); !errors.Is(err, actor.ErrPopulationNotFound) {
		t.Errorf("withdrawing a withdrawal: err = %v", err)
	}
	if _, err := um.WithdrawGroupPopulation(ctx, g.ID, "missing", "owner"); !errors.Is(err, actor.ErrPopulationNotFound) {
		t.Errorf("unknown figure: err = %v", err)
	}
	if _, err := um.LatestGroupPopulation(ctx, g.ID); !errors.Is(err, actor.ErrPopulationNotFound) {
		t.Errorf("latest with nothing standing: err = %v", err)
	}
}

func TestAGroupWithNoFigureTakesTheSumOfItsInnerGroupsAtTheOldestDate(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	district := groupNamed(t, um, "Kerület", "")
	north := groupNamed(t, um, "Észak", district.ID)
	south := groupNamed(t, um, "Dél", district.ID)
	groupNamed(t, um, "Üres", district.ID)
	um.RecordGroupPopulation(ctx, north.ID, 120, "2026-03-01", "census", "owner")
	um.RecordGroupPopulation(ctx, south.ID, 80, "2025-09-01", "census", "owner")
	sum, err := um.LatestGroupPopulation(ctx, district.ID)
	if err != nil {
		t.Fatalf("LatestGroupPopulation: %v", err)
	}
	if sum.Population != 200 || sum.AsOf != "2025-09-01" || sum.Source != "inner groups" || sum.GroupID != district.ID {
		t.Fatalf("sum = %+v, want 200 as of 2025-09-01 from inner groups", sum)
	}
}

func TestAGroupWithOwnFigureIgnoresItsInnerGroups(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	district := groupNamed(t, um, "Kerület", "")
	north := groupNamed(t, um, "Észak", district.ID)
	um.RecordGroupPopulation(ctx, north.ID, 120, "2026-03-01", "census", "owner")
	um.RecordGroupPopulation(ctx, district.ID, 500, "2026-01-01", "census", "owner")
	got, err := um.LatestGroupPopulation(ctx, district.ID)
	if err != nil || got.Population != 500 || got.Source != "census" {
		t.Fatalf("latest = %+v, %v; want its own 500", got, err)
	}
}
