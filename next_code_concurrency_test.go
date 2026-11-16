package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type statementCounter struct {
	logger.Interface
	statements atomic.Int64
}

func (c *statementCounter) LogMode(logger.LogLevel) logger.Interface { return c }

func (c *statementCounter) Trace(_ context.Context, _ time.Time, _ func() (string, int64), _ error) {
	c.statements.Add(1)
}

func openCounterDB(t *testing.T) (*gorm.DB, *statementCounter, string) {
	t.Helper()
	counter := &statementCounter{Interface: logger.Discard}
	dsn := "file:" + filepath.Join(t.TempDir(), "codes.db") + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(30000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: counter})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	return db, counter, tableFor(s, roleCodeSequence)
}

func TestFiftyConcurrentNextCodesAreUniqueWithoutConflict(t *testing.T) {
	const claimers = 50
	db, counter, table := openCounterDB(t)

	var wg sync.WaitGroup
	codes := make([]string, claimers)
	errs := make([]error, claimers)
	before := counter.statements.Load()
	for i := 0; i < claimers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], errs[i] = nextCode(context.Background(), db, table, "", "actor", "AC")
		}(i)
	}
	wg.Wait()
	used := counter.statements.Load() - before

	seen := map[string]bool{}
	for i, code := range codes {
		if errors.Is(errs[i], ErrConflict) {
			t.Fatalf("claimer %d returned a conflict: %v", i, errs[i])
		}
		if errs[i] != nil {
			t.Fatalf("claimer %d: %v", i, errs[i])
		}
		if seen[code] {
			t.Fatalf("code %s was handed out twice", code)
		}
		seen[code] = true
	}
	if limit := int64(2 * claimers); used > limit {
		t.Fatalf("%d claims took %d statements, want at most %d", claimers, used, limit)
	}
}

func TestNextCodeCountsFromOneAndKeepsEachTypeApart(t *testing.T) {
	db, _, table := openCounterDB(t)
	ctx := context.Background()

	want := []struct{ entityType, prefix, code string }{
		{"actor", "AC", "AC-1"},
		{"actor", "AC", "AC-2"},
		{"group", "G", "G-1"},
		{"actor", "AC", "AC-3"},
		{"group", "G", "G-2"},
	}
	for _, w := range want {
		got, err := nextCode(ctx, db, table, "", w.entityType, w.prefix)
		if err != nil {
			t.Fatalf("nextCode %s: %v", w.entityType, err)
		}
		if got != w.code {
			t.Fatalf("nextCode %s = %s, want %s", w.entityType, got, w.code)
		}
	}
}

func TestNextCodeCountsEachRunApart(t *testing.T) {
	db, _, _ := openCounterDB(t)
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	um, err := NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	ctx := context.Background()
	for _, run := range []string{"run-a", "run-b"} {
		inRunManager, err := um.ForRun(run)
		if err != nil {
			t.Fatalf("ForRun %s: %v", run, err)
		}
		for want := 1; want <= 2; want++ {
			created, err := inRunManager.CreateActor(ctx, Actor{DisplayName: "Kovács Anna"})
			if err != nil {
				t.Fatalf("CreateActor in %s: %v", run, err)
			}
			if wantCode := fmt.Sprintf("AC-%d", want); created.Code != wantCode {
				t.Fatalf("actor code in %s = %s, want %s", run, created.Code, wantCode)
			}
		}
	}
}
