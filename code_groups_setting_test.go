package mwanachamaactor_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func codeGroupsFixture(t *testing.T) (*gorm.DB, *spec.Spec, mwanachamaactor.UserManager) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	return db, s, newTestManagerOn(t, db)
}

func TestTheBlueprintDeclaresCodeGroupsDefaultingToToday(t *testing.T) {
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	st, ok := s.Setting("code_groups")
	if !ok {
		t.Fatal("code_groups is not declared")
	}
	n, err := models.ParseCodeGroups(st.Default)
	if err != nil || n != models.DefaultCodeGroups {
		t.Fatalf("default = %v (%v), want %d", st.Default, err, models.DefaultCodeGroups)
	}
}

func TestADefinitionValueSetsTheLengthOfNewCodes(t *testing.T) {
	db, s, mgr := codeGroupsFixture(t)
	ctx := context.Background()
	if err := spec.PutDefinitionSettings(db, s, map[string]any{"code_groups": 2}); err != nil {
		t.Fatalf("PutDefinitionSettings: %v", err)
	}
	f, err := mgr.CodeFormat(ctx)
	if err != nil || f.GroupCount != 2 {
		t.Fatalf("CodeFormat = %+v, %v; want 2 groups", f, err)
	}
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Ami"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if !publicCodeShape("M", 2).MatchString(a.PublicCode) {
		t.Fatalf("code %q is not two groups", a.PublicCode)
	}
}

func TestAnOverrideWinsOverTheDefinition(t *testing.T) {
	db, s, mgr := codeGroupsFixture(t)
	ctx := context.Background()
	if err := spec.PutDefinitionSettings(db, s, map[string]any{"code_groups": 2}); err != nil {
		t.Fatalf("PutDefinitionSettings: %v", err)
	}
	if _, err := spec.StoreOverride(db, s, "code_groups", 5, "op", mwanachamaactor.NowRFC3339()); err != nil {
		t.Fatalf("StoreOverride: %v", err)
	}
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if !publicCodeShape("G", 5).MatchString(g.PublicCode) {
		t.Fatalf("code %q is not five groups", g.PublicCode)
	}
}

func TestASetCodeFormatStillWinsOverTheSetting(t *testing.T) {
	db, s, mgr := codeGroupsFixture(t)
	ctx := context.Background()
	if _, err := spec.StoreOverride(db, s, "code_groups", 5, "op", mwanachamaactor.NowRFC3339()); err != nil {
		t.Fatalf("StoreOverride: %v", err)
	}
	if _, err := mgr.SetCodeFormat(ctx, 4); err != nil {
		t.Fatalf("SetCodeFormat: %v", err)
	}
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Ami"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if !publicCodeShape("M", 4).MatchString(a.PublicCode) {
		t.Fatalf("code %q is not four groups", a.PublicCode)
	}
}

func TestAnOverrideOutsideTheBoundsIsRefusedWhenMinting(t *testing.T) {
	db, s, mgr := codeGroupsFixture(t)
	if _, err := spec.StoreOverride(db, s, "code_groups", 40, "op", mwanachamaactor.NowRFC3339()); err != nil {
		t.Fatalf("StoreOverride: %v", err)
	}
	if _, err := mgr.CreateActor(context.Background(), mwanachamaactor.Actor{DisplayName: "Ami"}); err == nil {
		t.Fatal("a 40-group code was minted")
	}
}
