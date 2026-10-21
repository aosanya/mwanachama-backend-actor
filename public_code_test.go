package mwanachamaactor_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

const publicCodeChars = "[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{3}"

func publicCodeShape(prefix string, groups int) *regexp.Regexp {
	return regexp.MustCompile("^" + prefix + strings.Repeat("-"+publicCodeChars, groups) + "$")
}

func TestANewActorAndGroupDrawRandomCodesOfThreeGroups(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Ami"})
		if err != nil {
			t.Fatalf("CreateActor: %v", err)
		}
		if !publicCodeShape("M", 3).MatchString(a.PublicCode) {
			t.Fatalf("actor public code = %q, want M- and three groups with no look-alikes", a.PublicCode)
		}
		if !strings.HasPrefix(a.Code, "AC-") {
			t.Errorf("internal code = %q, want AC-<n> kept", a.Code)
		}
		g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward"})
		if err != nil {
			t.Fatalf("CreateGroup: %v", err)
		}
		if !publicCodeShape("G", 3).MatchString(g.PublicCode) {
			t.Fatalf("group public code = %q, want G- and three groups with no look-alikes", g.PublicCode)
		}
		for _, code := range []string{a.PublicCode, g.PublicCode} {
			if seen[code] {
				t.Fatalf("code %s drawn twice", code)
			}
			seen[code] = true
		}
		read, err := mgr.GetActor(ctx, a.ID)
		if err != nil || read.PublicCode != a.PublicCode {
			t.Fatalf("stored actor = %+v, %v; want public code %s", read, err, a.PublicCode)
		}
	}
}

func TestTheCodeFormatStartsAtThreeAndAppliesToCodesDrawnAfterwards(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	format, err := mgr.CodeFormat(ctx)
	if err != nil || format.GroupCount != 3 {
		t.Fatalf("CodeFormat = %+v, %v; want three groups", format, err)
	}
	before, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}

	set, err := mgr.SetCodeFormat(ctx, 5)
	if err != nil || set.GroupCount != 5 {
		t.Fatalf("SetCodeFormat = %+v, %v; want five groups", set, err)
	}
	if again, err := mgr.SetCodeFormat(ctx, 4); err != nil || again.GroupCount != 4 {
		t.Fatalf("SetCodeFormat again = %+v, %v; want four groups", again, err)
	}
	if read, err := mgr.CodeFormat(ctx); err != nil || read.GroupCount != 4 || read.LastUpdated == "" {
		t.Fatalf("CodeFormat after setting = %+v, %v; want four groups", read, err)
	}

	after, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if !publicCodeShape("M", 4).MatchString(after.PublicCode) {
		t.Errorf("a code drawn after the change = %q, want four groups", after.PublicCode)
	}
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if !publicCodeShape("G", 4).MatchString(g.PublicCode) {
		t.Errorf("a group code drawn after the change = %q, want four groups", g.PublicCode)
	}
	kept, err := mgr.GetActor(ctx, before.ID)
	if err != nil || kept.PublicCode != before.PublicCode {
		t.Errorf("an earlier code = %+v, %v; want %s kept", kept, err, before.PublicCode)
	}
}

func TestACodeFormatOutsideItsRangeIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	for _, n := range []int{0, -1, 13} {
		if _, err := mgr.SetCodeFormat(context.Background(), n); !errors.Is(err, mwanachamaactor.ErrInvalidCodeFormat) {
			t.Errorf("SetCodeFormat(%d) err = %v, want ErrInvalidCodeFormat", n, err)
		}
	}
}

func TestProvisionGivesEveryExistingActorAndGroupAPublicCode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	ctx := context.Background()
	var actors []mwanachamaactor.Actor
	var groups []mwanachamaactor.Group
	for i := 0; i < 5; i++ {
		a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{})
		if err != nil {
			t.Fatalf("CreateActor: %v", err)
		}
		actors = append(actors, a)
		g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward", Hidden: i%2 == 0})
		if err != nil {
			t.Fatalf("CreateGroup: %v", err)
		}
		groups = append(groups, g)
	}
	for _, role := range []string{"actor", "group"} {
		o, _ := s.ByRole(role)
		for _, idx := range o.Indexes {
			if idx.Name == "public_code" {
				if err := db.Exec("drop index " + s.IndexFor(o, idx)).Error; err != nil {
					t.Fatalf("drop the %s public code index: %v", role, err)
				}
			}
		}
		if err := db.Exec("alter table " + s.TableFor(o) + " drop column public_code").Error; err != nil {
			t.Fatalf("drop %s.public_code: %v", role, err)
		}
	}

	again := newTestManagerOn(t, db)
	seen := map[string]bool{}
	for _, a := range actors {
		got, err := again.GetActor(ctx, a.ID)
		if err != nil {
			t.Fatalf("GetActor: %v", err)
		}
		if !publicCodeShape("M", 3).MatchString(got.PublicCode) || seen[got.PublicCode] || got.Code != a.Code {
			t.Errorf("backfilled actor = %+v, want a fresh M- code and %s kept", got, a.Code)
		}
		seen[got.PublicCode] = true
	}
	for _, g := range groups {
		got, err := again.GetGroup(ctx, g.ID)
		if err != nil {
			t.Fatalf("GetGroup: %v", err)
		}
		if !publicCodeShape("G", 3).MatchString(got.PublicCode) || seen[got.PublicCode] || got.Code != g.Code {
			t.Errorf("backfilled group = %+v, want a fresh G- code and %s kept", got, g.Code)
		}
		seen[got.PublicCode] = true
	}

	stable := newTestManagerOn(t, db)
	got, err := stable.GetActor(ctx, actors[0].ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if want, _ := again.GetActor(ctx, actors[0].ID); got.PublicCode != want.PublicCode {
		t.Errorf("a second provision redrew %s as %s", want.PublicCode, got.PublicCode)
	}
}
