// postgres_integration_test.go exercises UserManager against a real
// Postgres-backed entitygraph.DataManager (mwanachama-backend-shared's
// postgres.Backend), rather than the in-memory fakeDataManager the rest of
// this package's tests use.
//
// Skipped unless POSTGRES_URL is set — mirrors
// mwanachama-backend-taskmanager's own postgres_integration_test.go. The unit
// tests elsewhere in this package already exhaustively cover UserManager's
// business logic against fakeDataManager; this file's job is narrower —
// prove the real Postgres wiring (schema activation, jsonb property
// round-trips, relationship storage) works end-to-end.
package mwanachamaactor_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/postgres"
	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func applyDDL(ctx context.Context, db *sql.DB, script string) error {
	_, err := db.ExecContext(ctx, script)
	return err
}

// newPostgresUserManager opens POSTGRES_URL, creates a scratch set of
// member-prefixed tables, seeds+activates DefaultUserSchema, and returns a
// ready-to-use UserManager. Skips the calling test if POSTGRES_URL is unset.
// Tables are dropped on cleanup.
func newPostgresUserManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test")
	}

	ctx := context.Background()
	db, err := postgres.Open(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// A unique-enough prefix per test keeps concurrent -run invocations from
	// colliding on the same physical tables.
	tables := postgres.DefaultTableNames("useri_")
	if err := applyDDL(ctx, db, postgres.DDL(tables)); err != nil {
		t.Fatalf("applying DDL: %v", err)
	}
	t.Cleanup(func() {
		_ = applyDDL(context.Background(), db, postgres.DropDDL(tables))
	})

	backend := postgres.NewBackend(db, tables)

	s := mwanachamaactor.DefaultUserSchema("useri")
	if err := backend.SetSchema(ctx, s); err != nil {
		t.Fatalf("SetSchema: %v", err)
	}
	if err := backend.Publish(ctx); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := backend.Activate(ctx, 1); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	mgr, err := mwanachamaactor.NewUserManager(backend)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func TestPostgres_MemberCRUD_RoundTrip(t *testing.T) {
	mgr := newPostgresUserManager(t)
	ctx := context.Background()

	created, err := mgr.CreateMember(ctx, mwanachamaactor.Member{
		DisplayName: "Postgres round-trip",
		Email:       "pg@example.com",
		Attributes:  map[string]any{"persona": "trader"},
	})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}

	got, err := mgr.GetMember(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.DisplayName != "Postgres round-trip" || got.Email != "pg@example.com" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.Attributes["persona"] != "trader" {
		t.Errorf("Attributes = %v, want persona=trader (jsonb round-trip)", got.Attributes)
	}

	if _, err := mgr.SetMemberDisplayName(ctx, created.ID, "Renamed"); err != nil {
		t.Fatalf("SetMemberDisplayName: %v", err)
	}
	list, err := mgr.ListMembers(ctx)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(list) != 1 || list[0].DisplayName != "Renamed" {
		t.Errorf("ListMembers = %+v", list)
	}
}

func TestPostgres_GroupAndRegistration_RoundTrip(t *testing.T) {
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

	m, err := mgr.CreateMember(ctx, mwanachamaactor.Member{DisplayName: "Member"})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}
	if _, err := mgr.Register(ctx, mwanachamaactor.Registration{
		MemberID: m.ID, GroupID: child.ID, IsHome: true,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	regs, err := mgr.ListGroupsForMember(ctx, m.ID)
	if err != nil {
		t.Fatalf("ListGroupsForMember: %v", err)
	}
	if len(regs) != 1 || regs[0].GroupID != child.ID {
		t.Errorf("ListGroupsForMember = %+v", regs)
	}

	counts, err := mgr.HomeCounts(ctx)
	if err != nil {
		t.Fatalf("HomeCounts: %v", err)
	}
	if counts[child.ID] != 1 {
		t.Errorf("HomeCounts[child] = %d, want 1", counts[child.ID])
	}

	gone, found, err := mgr.Deregister(ctx, m.ID, child.ID)
	if err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	if !found || !gone.IsHome {
		t.Errorf("Deregister result = found=%v gone=%+v", found, gone)
	}
}
