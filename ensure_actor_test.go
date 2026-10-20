package mwanachamaactor_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestEnsureActor_CreatesTheActorUnderTheGivenID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	got, err := mgr.EnsureActor(ctx, "c0ffee00-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatalf("EnsureActor: %v", err)
	}
	if got.ID != "c0ffee00-0000-4000-8000-000000000001" {
		t.Fatalf("id = %q, want the given id", got.ID)
	}
	if got.Code == "" || got.CreatedAt == "" {
		t.Fatalf("expected a minted code and timestamp, got %+v", got)
	}
	read, err := mgr.GetActor(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if read.Code != got.Code {
		t.Fatalf("stored code %q, returned %q", read.Code, got.Code)
	}
}

func TestEnsureActor_ASecondCallReturnsTheSameActor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	first, err := mgr.EnsureActor(ctx, "user-1")
	if err != nil {
		t.Fatalf("first EnsureActor: %v", err)
	}
	second, err := mgr.EnsureActor(ctx, "user-1")
	if err != nil {
		t.Fatalf("second EnsureActor: %v", err)
	}
	if second.Code != first.Code || second.CreatedAt != first.CreatedAt {
		t.Fatalf("second call changed the actor: first %+v, second %+v", first, second)
	}
	all, err := mgr.ListActors(ctx)
	if err != nil {
		t.Fatalf("ListActors: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected one actor, got %d", len(all))
	}
}

func TestEnsureActor_RefusesAnEmptyID(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.EnsureActor(context.Background(), "  "); !errors.Is(err, mwanachamaactor.ErrInvalidActor) {
		t.Fatalf("expected ErrInvalidActor, got %v", err)
	}
}

func TestEnsureActor_LetsAnAssignmentNameTheGivenID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	group, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: "user-2", GroupID: group.ID}); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Fatalf("before EnsureActor, expected ErrActorNotFound, got %v", err)
	}
	if _, err := mgr.EnsureActor(ctx, "user-2"); err != nil {
		t.Fatalf("EnsureActor: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: "user-2", GroupID: group.ID}); err != nil {
		t.Fatalf("AssignGroup after EnsureActor: %v", err)
	}
}

func TestEnsureActor_TwoConcurrentFirstCallsBothSucceedWithOneActor(t *testing.T) {
	dsn := fmt.Sprintf("%s/ensure.db?_pragma=busy_timeout(5000)", t.TempDir())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaactor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaactor.NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	ctx := context.Background()
	if _, err := mgr.EnsureActor(ctx, "warm-up"); err != nil {
		t.Fatalf("warm-up EnsureActor: %v", err)
	}

	actorObject, ok := s.ByRole("actor")
	if !ok {
		t.Fatal("spec declares no actor role")
	}
	actors := s.TableFor(actorObject)
	var arrived sync.WaitGroup
	arrived.Add(2)
	var held atomic.Int32
	db.Callback().Query().After("gorm:query").Register("ensure_barrier", func(tx *gorm.DB) {
		if tx.Statement.Table != actors || held.Add(1) > 2 {
			return
		}
		arrived.Done()
		arrived.Wait()
	})

	var errs [2]error
	var results [2]mwanachamaactor.Actor
	var done sync.WaitGroup
	done.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer done.Done()
			results[i], errs[i] = mgr.EnsureActor(ctx, "user-raced")
		}()
	}
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d failed: %v", i, err)
		}
	}
	if results[0].Code != results[1].Code {
		t.Fatalf("the two callers saw different actors: %+v and %+v", results[0], results[1])
	}
	all, err := mgr.ListActors(ctx)
	if err != nil {
		t.Fatalf("ListActors: %v", err)
	}
	count := 0
	for _, a := range all {
		if a.ID == "user-raced" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one raced actor, got %d", count)
	}
}
