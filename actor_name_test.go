package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestAnActorsFirstAndLastNameAreSetBesideTheDisplayName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Ami"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if a.FirstName != "" || a.LastName != "" {
		t.Fatalf("a new actor carries a name: %+v", a)
	}

	named, err := mgr.SetActorName(ctx, a.ID, "Amina", "Wanjiru")
	if err != nil {
		t.Fatalf("SetActorName: %v", err)
	}
	if named.FirstName != "Amina" || named.LastName != "Wanjiru" || named.DisplayName != "Ami" {
		t.Errorf("SetActorName answered %+v", named)
	}
	read, err := mgr.GetActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if read.FirstName != "Amina" || read.LastName != "Wanjiru" || read.DisplayName != "Ami" || read.Code != a.Code {
		t.Errorf("GetActor read %+v", read)
	}

	if _, err := mgr.SetActorDisplayName(ctx, a.ID, "Amie"); err != nil {
		t.Fatalf("SetActorDisplayName: %v", err)
	}
	read, _ = mgr.GetActor(ctx, a.ID)
	if read.FirstName != "Amina" || read.LastName != "Wanjiru" {
		t.Errorf("renaming the display name moved the name: %+v", read)
	}

	if _, err := mgr.SetActorName(ctx, a.ID, "Amina", ""); err != nil {
		t.Fatalf("SetActorName clearing the last name: %v", err)
	}
	read, _ = mgr.GetActor(ctx, a.ID)
	if read.LastName != "" || read.FirstName != "Amina" || read.DisplayName != "Amie" {
		t.Errorf("clearing the last name read back %+v", read)
	}
}

func TestSetActorNameRefusesAnUnknownActor(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.SetActorName(context.Background(), "nope", "A", "B"); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Fatalf("err = %v, want ErrActorNotFound", err)
	}
}

func TestProvisionAddsTheNameColumnsToAnExistingActorTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	a, err := mgr.CreateActor(context.Background(), mwanachamaactor.Actor{DisplayName: "Ami"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	o, _ := s.ByRole("actor")
	for _, column := range []string{"first_name", "last_name"} {
		if err := db.Exec("alter table " + s.TableFor(o) + " drop column " + column).Error; err != nil {
			t.Fatalf("drop %s: %v", column, err)
		}
	}

	again := newTestManagerOn(t, db)
	got, err := again.GetActor(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if got.FirstName != "" || got.LastName != "" || got.DisplayName != "Ami" {
		t.Errorf("an existing actor read back %+v", got)
	}
	if _, err := again.SetActorName(context.Background(), a.ID, "Amina", "Wanjiru"); err != nil {
		t.Errorf("SetActorName on the restored columns: %v", err)
	}
}
