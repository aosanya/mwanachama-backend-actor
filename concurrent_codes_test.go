package mwanachamaactor_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func newSharedCacheManager(t *testing.T, name string) (mwanachamaactor.UserManager, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() { sqlDB.Close() })
	return newTestManagerOn(t, db), db
}

type raceOutcome struct {
	code string
	err  error
}

func raceCreates(writers int, create func(i int) (string, error)) []raceOutcome {
	outcomes := make([]raceOutcome, writers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			code, err := create(i)
			outcomes[i] = raceOutcome{code: code, err: err}
		}()
	}
	close(start)
	wg.Wait()
	return outcomes
}

func requireDistinctCodesAndClassifiedLosers(t *testing.T, round int, outcomes []raceOutcome) {
	t.Helper()
	seen := map[string]bool{}
	defer func() {
		if len(seen) == 0 {
			t.Errorf("round %d: no writer minted a code", round)
		}
	}()
	for _, o := range outcomes {
		if o.err != nil {
			if !errors.Is(o.err, mwanachamaactor.ErrConflict) {
				t.Errorf("round %d: a losing writer saw %v, want ErrConflict", round, o.err)
			}
			continue
		}
		if seen[o.code] {
			t.Fatalf("round %d: two writers were handed the code %q", round, o.code)
		}
		seen[o.code] = true
	}
}

func TestACT3_ConcurrentCreateActorMintsDistinctCodesAndClassifiesTheRest(t *testing.T) {
	mgr, _ := newSharedCacheManager(t, "act3actor")
	ctx := context.Background()
	for round := 0; round < 10; round++ {
		outcomes := raceCreates(8, func(int) (string, error) {
			a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{})
			return a.Code, err
		})
		requireDistinctCodesAndClassifiedLosers(t, round, outcomes)
	}
}

func TestACT3_ConcurrentCreateHierarchyMintsDistinctCodesAndClassifiesTheRest(t *testing.T) {
	mgr, _ := newSharedCacheManager(t, "act3hierarchy")
	ctx := context.Background()
	for round := 0; round < 10; round++ {
		outcomes := raceCreates(6, func(i int) (string, error) {
			h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: fmt.Sprintf("H%d-%d", round, i)})
			return h.Code, err
		})
		requireDistinctCodesAndClassifiedLosers(t, round, outcomes)
	}
}

func TestACT3_ConcurrentCreateRoleKindMintsDistinctCodesAndClassifiesTheRest(t *testing.T) {
	mgr, _ := newSharedCacheManager(t, "act3rolekind")
	ctx := context.Background()
	for round := 0; round < 10; round++ {
		outcomes := raceCreates(6, func(i int) (string, error) {
			k, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: fmt.Sprintf("K%d-%d", round, i)})
			return k.Code, err
		})
		requireDistinctCodesAndClassifiedLosers(t, round, outcomes)
	}
}

func TestACT3_ConcurrentCreateGroupMintsDistinctCodesAndClassifiesTheRest(t *testing.T) {
	mgr, _ := newSharedCacheManager(t, "act3group")
	ctx := context.Background()
	h, err := mgr.CreateHierarchy(ctx, mwanachamaactor.Hierarchy{Name: "Governance"})
	if err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}
	l, err := mgr.CreateLevel(ctx, mwanachamaactor.Level{HierarchyID: h.ID, Name: "Branch"})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	for round := 0; round < 10; round++ {
		outcomes := raceCreates(6, func(i int) (string, error) {
			g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{HierarchyID: h.ID, LevelID: l.ID, Name: fmt.Sprintf("G%d-%d", round, i)})
			return g.Code, err
		})
		requireDistinctCodesAndClassifiedLosers(t, round, outcomes)
	}
}
