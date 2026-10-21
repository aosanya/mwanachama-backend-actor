package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestEveryActorStartsWithTheFormsWeekSplitOn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	created, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	ensured, err := mgr.EnsureActor(ctx, "caller-1")
	if err != nil {
		t.Fatalf("EnsureActor: %v", err)
	}
	for _, id := range []string{created.ID, ensured.ID} {
		got, err := mgr.GetMyFormsWeekSplit(ctx, id)
		if err != nil || !got.FormsWeekSplit || got.ActorID != id {
			t.Errorf("GetMyFormsWeekSplit(%s) = %+v, %v, want on", id, got, err)
		}
		read, _ := mgr.GetActor(ctx, id)
		if !read.FormsWeekSplit {
			t.Errorf("GetActor(%s).FormsWeekSplit = false, want true", id)
		}
	}
	if got, err := mgr.GetMyFormsWeekSplit(ctx, "no-record"); err != nil || !got.FormsWeekSplit {
		t.Errorf("an actor with no record = %+v, %v, want on", got, err)
	}
}

func TestAnActorTurnsTheirOwnFormsWeekSplitOffAndOn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	b, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	for _, want := range []bool{false, true, false} {
		got, err := mgr.SetMyFormsWeekSplit(ctx, a.ID, want)
		if err != nil || got.FormsWeekSplit != want {
			t.Fatalf("SetMyFormsWeekSplit(%v) = %+v, %v", want, got, err)
		}
		read, _ := mgr.GetMyFormsWeekSplit(ctx, a.ID)
		if read.FormsWeekSplit != want {
			t.Errorf("read = %v, want %v", read.FormsWeekSplit, want)
		}
	}
	if got, _ := mgr.GetMyFormsWeekSplit(ctx, b.ID); !got.FormsWeekSplit {
		t.Errorf("another actor's split = off, want on untouched")
	}
	if _, err := mgr.SetActorName(ctx, a.ID, "Amina", "Wanjiru"); err != nil {
		t.Fatalf("SetActorName: %v", err)
	}
	if got, _ := mgr.GetMyFormsWeekSplit(ctx, a.ID); got.FormsWeekSplit {
		t.Errorf("split after a name change = on, want off kept")
	}
}

func TestSettingTheFormsWeekSplitRefusesNoCallerAndNoRecord(t *testing.T) {
	mgr := newTestManager(t)
	for _, id := range []string{"", "no-record"} {
		if _, err := mgr.SetMyFormsWeekSplit(context.Background(), id, false); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
			t.Errorf("SetMyFormsWeekSplit(%q) err = %v, want ErrActorNotFound", id, err)
		}
	}
}

func TestProvisionAddsTheFormsWeekSplitOnToAnExistingActorTable(t *testing.T) {
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
	if err := db.Exec("alter table " + s.TableFor(o) + " drop column forms_week_split").Error; err != nil {
		t.Fatalf("drop forms_week_split: %v", err)
	}

	again := newTestManagerOn(t, db)
	got, err := again.GetActor(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if !got.FormsWeekSplit {
		t.Errorf("an actor older than the column read back %+v, want the split on", got)
	}
	if _, err := again.SetMyFormsWeekSplit(context.Background(), a.ID, false); err != nil {
		t.Errorf("SetMyFormsWeekSplit on the restored column: %v", err)
	}
}
