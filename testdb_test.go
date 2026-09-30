package mwanachamaactor_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func newTestManager(t *testing.T) mwanachamaactor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return newTestManagerOn(t, db)
}

func newTestManagerOn(t *testing.T, db *gorm.DB) mwanachamaactor.UserManager {
	t.Helper()
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
	return mgr
}

func TestNewUserManager_NilDB(t *testing.T) {
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if _, err := mwanachamaactor.NewUserManager(nil, s); err == nil {
		t.Fatal("expected error for nil db")
	}
}
