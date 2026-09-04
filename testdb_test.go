package mwanachamaactor_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

// newTestManager builds a [mwanachamaactor.UserManager] backed by a fresh
// in-memory sqlite database, migrated the same way a real deployment would
// via [mwanachamaactor.Migrate]. Replaces the old hand-maintained
// fakeDataManager: exercising real GORM/SQL behavior catches more than a Go
// map fake ever could, while staying fully in-process — no containers, no
// POSTGRES_URL, consistent with this repo's existing separation between fast
// unit tests here and the opt-in postgres_integration_test.go.
func newTestManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	tables := mwanachamaactor.DefaultTableNames("test")
	if err := mwanachamaactor.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	mgr, err := mwanachamaactor.NewUserManager(db, tables)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func TestNewUserManager_NilDB(t *testing.T) {
	if _, err := mwanachamaactor.NewUserManager(nil, mwanachamaactor.DefaultTableNames("test")); err == nil {
		t.Fatal("expected error for nil db")
	}
}
