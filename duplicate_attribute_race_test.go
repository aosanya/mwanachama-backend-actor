package mwanachamaactor_test

// Pins board row ACT1 (documentation/3. implementation/todo.md): the
// DB-level partial unique index that backstops checkUniqueAttributes'
// Go-level pre-check (gormstore.syncUniqueAttributeIndexes,
// "actors_attr_phone_uniq") DOES stop two concurrent CreateActor calls with
// the same phone from both succeeding — that part works. But the resulting
// error is misclassified: actor_impl.go's classifyDuplicateID treats ANY
// unique-constraint violation on the actors table (SQLSTATE 23505 /
// "UNIQUE constraint failed") as a primary-key collision and returns
// ErrDuplicateID, even when the violated index is the phone-attribute
// index, not the primary key. routes/actor.go's actorStatusFor has no case
// for ErrDuplicateID at all, so it falls to the default 500 arm — an
// outside HTTP caller who loses a same-phone race gets
// `500 {"error":"internal error"}` instead of the clean
// `400 {"error":"...attribute value already in use..."}` a *sequential*
// duplicate-phone request gets (caught by the Go-level precheck, which
// correctly returns ErrDuplicateAttribute).
//
// Once fixed (classifyDuplicateID should distinguish the attribute index by
// name/constraint rather than treating every unique violation as a PK
// collision, and/or actorStatusFor should map ErrDuplicateID to a non-500
// status), this test's assertions should flip: the raced call should surface
// errors.Is(err, ErrDuplicateAttribute) and the HTTP loser should get 400,
// not 500 — at which point this test goes red and should be updated to
// assert the fixed behavior.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

// newRaceTestManager is like testdb_test.go's newTestManager, but returns
// the raw *gorm.DB too (so a test can register a callback) and opens a
// real file-backed, WAL-mode sqlite database (under t.TempDir(), removed
// automatically) allowing genuine concurrent writers — newTestManager's
// private ":memory:" database and default rollback-journal mode both stand
// in the way of reproducing a real concurrent-write race deterministically,
// and an unnamed ":memory:"-with-"cache=shared" database uses sqlite's
// degraded shared-cache table locking, which spuriously
// "database table is locked: database is deadlocked"s under real
// concurrency rather than queuing the second writer the way a real WAL
// file (or Postgres, this package's actual deployment target) does — a
// sqlite-shared-cache testing artifact, not a finding about this package.
func newRaceTestManager(t *testing.T) (mwanachamaactor.UserManager, *gorm.DB, mwanachamaactor.TableNames) {
	t.Helper()
	dsn := fmt.Sprintf("%s/race.db?_pragma=busy_timeout(5000)", t.TempDir())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	// A single physical connection: our explicit Query-callback barrier
	// (raceBarrier, below) is what supplies the "two callers interleaved
	// before either committed" scenario we need, deterministically, so a
	// second real writer connection isn't needed to get genuine
	// concurrency — and serializing the actual SQL through one connection
	// sidesteps sqlite's shared-cache/WAL multi-writer locking edge cases
	// (SQLITE_BUSY/"database is locked") that are a sqlite testing-harness
	// artifact, not a finding about this package (its real deployment
	// target is Postgres).
	sqlDB.SetMaxOpenConns(1)
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if err := db.Exec("PRAGMA busy_timeout=5000").Error; err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}

	tables := mwanachamaactor.DefaultTableNames("test")
	if err := mwanachamaactor.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mgr, err := mwanachamaactor.NewUserManager(db, tables)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr, db, tables
}

// raceBarrier registers a Query callback that blocks each of the first two
// queries against tables.Actors until both have arrived, deterministically
// forcing two concurrent CreateActor calls' checkUniqueAttributes precheck
// to both actually complete — SQL sent, result (zero matching rows)
// already read back — before either goroutine is allowed to proceed to its
// own transaction/insert. Registered on the Query processor's After hook
// (not Before): a Before hook fires before gorm has even borrowed a
// connection from the pool, so with a single pooled connection
// (newRaceTestManager's SetMaxOpenConns(1)) synchronizing there only
// guarantees both goroutines *arrived*, not that both actually finished
// reading "zero rows" before either committed — After closes that gap, so
// the only way one goroutine's insert can still see the other's is via the
// DB-level partial unique index, not a scheduling accident of the test
// itself.
func raceBarrier(db *gorm.DB, actorsTable string) {
	var wg sync.WaitGroup
	wg.Add(2)
	db.Callback().Query().After("gorm:query").Register("race_barrier", func(tx *gorm.DB) {
		if tx.Statement.Table != actorsTable {
			return
		}
		wg.Done()
		wg.Wait()
	})
}

func TestCreateActor_PinsRacedDuplicatePhoneMisclassifiedAsID(t *testing.T) {
	mgr, db, tables := newRaceTestManager(t)
	ctx := context.Background()

	// Warm up the "actor" CodeSequence row sequentially first (its own
	// read-or-create-then-update, gormstore/codesequence.go's NextCode, has
	// no row lock either — racing a truly first-ever mint can itself
	// deadlock sqlite before the race we're isolating here even starts).
	// That's a related but separate gap from ACT1; not pinned by this test.
	if _, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		Attributes: map[string]any{"phone": "+15550007777"},
	}); err != nil {
		t.Fatalf("warm-up CreateActor: %v", err)
	}

	raceBarrier(db, tables.Actors)

	var errs [2]error
	var start sync.WaitGroup
	start.Add(1)
	var done sync.WaitGroup
	done.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer done.Done()
			start.Wait()
			_, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
				Attributes: map[string]any{"phone": "+15550009999"},
			})
			errs[i] = err
		}()
	}
	start.Done()
	done.Wait()

	successes, failures := 0, 0
	var loserErr error
	for _, err := range errs {
		if err == nil {
			successes++
		} else {
			failures++
			loserErr = err
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("expected exactly one success and one failure racing the same phone, got successes=%d failures=%d errs=%v", successes, failures, errs)
	}

	// CURRENT (broken) behavior: the DB-level partial unique index does
	// stop the duplicate — good — but the loser is misclassified as
	// ErrDuplicateID (an id collision) rather than ErrDuplicateAttribute
	// (the phone collision that actually happened).
	if !errors.Is(loserErr, mwanachamaactor.ErrDuplicateID) {
		t.Fatalf("expected the raced loser to be misclassified as ErrDuplicateID (pinning ACT1's current behavior), got: %v", loserErr)
	}
	if errors.Is(loserErr, mwanachamaactor.ErrDuplicateAttribute) {
		t.Fatalf("loser now correctly classified as ErrDuplicateAttribute — ACT1 appears fixed; update this test to assert the fixed behavior instead of the bug")
	}
}

func TestActorRoutes_PinsRacedDuplicatePhoneReturns500NotConflict(t *testing.T) {
	mgr, db, tables := newRaceTestManager(t)

	// Warm up the "actor" CodeSequence row first — see the sibling test's
	// identical comment for why.
	if _, err := mgr.CreateActor(context.Background(), mwanachamaactor.Actor{
		Attributes: map[string]any{"phone": "+15550006666"},
	}); err != nil {
		t.Fatalf("warm-up CreateActor: %v", err)
	}

	raceBarrier(db, tables.Actors)

	mux := http.NewServeMux()
	for _, rt := range routes.ActorRoutes(mgr, routes.ResourceNames{}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := []byte(`{"attributes":{"phone":"+15550008888"}}`)

	statuses := make([]int, 2)
	var start sync.WaitGroup
	start.Add(1)
	var done sync.WaitGroup
	done.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer done.Done()
			start.Wait()
			resp, err := http.Post(srv.URL+"/actors", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Errorf("POST /actors: %v", err)
				return
			}
			defer resp.Body.Close()
			statuses[i] = resp.StatusCode
		}()
	}
	start.Done()
	done.Wait()

	created, other := 0, 0
	var loserStatus int
	for _, s := range statuses {
		if s == http.StatusCreated {
			created++
		} else {
			other++
			loserStatus = s
		}
	}
	if created != 1 || other != 1 {
		t.Fatalf("expected exactly one 201 and one non-201 among the two racing HTTP requests, got statuses=%v", statuses)
	}

	// CURRENT (broken) behavior: the losing request gets an opaque 500,
	// because actorStatusFor (routes/actor.go) has no case for
	// ErrDuplicateID and falls through to its default arm. A caller
	// sending the identical duplicate-phone body sequentially (no race)
	// gets a clean 400 instead — see routes/actor_test.go for that case.
	// Once fixed, this should become http.StatusBadRequest (or another
	// clean 4xx), matching the sequential case.
	if loserStatus != http.StatusInternalServerError {
		t.Fatalf("expected the raced loser to get 500 (pinning ACT1's current behavior), got %d — ACT1 appears fixed; update this test to assert the fixed status code", loserStatus)
	}
}
