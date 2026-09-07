// postgres_integration_test.go exercises UserManager against a real
// Postgres database, rather than the in-memory sqlite-backed manager the
// rest of this package's tests use.
//
// Skipped unless POSTGRES_URL is set. The unit tests elsewhere in this
// package already exhaustively cover UserManager's business logic; this
// file's job is narrower — prove the real Postgres wiring (JSONB Attributes
// round-trips, GORM AutoMigrate) works end-to-end.
package mwanachamaactor_test

import (
	"context"
	"errors"
	"os"
	"testing"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-shared/postgres"
)

// newPostgresUserManager opens POSTGRES_URL via
// mwanachama-backend-shared/postgres.Open (the same DSN parsing, pgx
// driver, and pooling every other repo already uses), wraps that connection
// with GORM's Postgres dialector, migrates a unique-enough table prefix, and
// returns a ready-to-use UserManager. Skips the calling test if
// POSTGRES_URL is unset. Tables are dropped on cleanup.
func newPostgresUserManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test")
	}

	ctx := context.Background()
	sqlDB, err := postgres.Open(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("postgres.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	// A unique-enough prefix per test keeps concurrent -run invocations from
	// colliding on the same physical tables.
	tables := mwanachamaactor.DefaultTableNames("useri")
	if err := mwanachamaactor.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(tables.ActorGroupAssignments, tables.Actors, tables.Groups)
	})

	mgr, err := mwanachamaactor.NewUserManager(db, tables)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func TestPostgres_ActorCRUD_RoundTrip(t *testing.T) {
	mgr := newPostgresUserManager(t)
	ctx := context.Background()

	created, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "Postgres round-trip",
		Attributes:  map[string]any{"email": "pg@example.com", "persona": "trader"},
	})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}

	got, err := mgr.GetActor(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if got.DisplayName != "Postgres round-trip" || got.Attributes["email"] != "pg@example.com" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.Attributes["persona"] != "trader" {
		t.Errorf("Attributes = %v, want persona=trader (jsonb round-trip)", got.Attributes)
	}

	if _, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "Postgres duplicate email",
		Attributes:  map[string]any{"email": "pg@example.com"},
	}); !errors.Is(err, mwanachamaactor.ErrDuplicateAttribute) {
		t.Errorf("CreateActor(duplicate email) err = %v, want ErrDuplicateAttribute", err)
	}

	if _, err := mgr.SetActorDisplayName(ctx, created.ID, "Renamed"); err != nil {
		t.Fatalf("SetActorDisplayName: %v", err)
	}
	list, err := mgr.ListActors(ctx)
	if err != nil {
		t.Fatalf("ListActors: %v", err)
	}
	if len(list) != 1 || list[0].DisplayName != "Renamed" {
		t.Errorf("ListActors = %+v", list)
	}
}

func TestPostgres_GroupAndActorGroupAssignment_RoundTrip(t *testing.T) {
	mgr := newPostgresUserManager(t)
	ctx := context.Background()

	root, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Root", Discoverable: true})
	if err != nil {
		t.Fatalf("CreateGroup(root): %v", err)
	}
	child, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Child", ParentID: root.ID})
	if err != nil {
		t.Fatalf("CreateGroup(child): %v", err)
	}

	kids, err := mgr.ListGroupChildren(ctx, root.ID)
	if err != nil {
		t.Fatalf("ListGroupChildren: %v", err)
	}
	if len(kids) != 1 || kids[0].ID != child.ID {
		t.Errorf("ListGroupChildren = %+v", kids)
	}

	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Actor"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{
		ActorID: a.ID, GroupID: child.ID, Attributes: map[string]any{"is_home": true},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	regs, err := mgr.ListGroupsForActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	if len(regs) != 1 || regs[0].GroupID != child.ID {
		t.Errorf("ListGroupsForActor = %+v", regs)
	}

	counts, err := mgr.HomeCounts(ctx)
	if err != nil {
		t.Fatalf("HomeCounts: %v", err)
	}
	if counts[child.ID] != 1 {
		t.Errorf("HomeCounts[child] = %d, want 1", counts[child.ID])
	}

	gone, found, err := mgr.Deregister(ctx, a.ID, child.ID)
	if err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	isHome, _ := gone.Attributes["is_home"].(bool)
	if !found || !isHome {
		t.Errorf("Deregister result = found=%v gone=%+v", found, gone)
	}
}
