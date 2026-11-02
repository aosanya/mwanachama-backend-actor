package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (f *fixture) intoWAL(t *testing.T) {
	t.Helper()
	if err := f.db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		t.Fatalf("journal_mode=WAL: %v", err)
	}
	f.duplicate(t)
	sqlDB, _ := f.db.DB()
	sqlDB.Close()
}

func entryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestAWALDatabaseIsReadWithoutChangingItsFile(t *testing.T) {
	f := newFixture(t, "")
	f.intoWAL(t)
	before := fileState(t, f.path)

	got := f.run()

	if got.code != 1 {
		t.Fatalf("outcome = %+v, want exit 1", got)
	}
	if after := fileState(t, f.path); after != before {
		t.Fatalf("the database file changed: before %+v, after %+v", before, after)
	}
	for _, name := range entryNames(t, filepath.Dir(f.path)) {
		if name != "actor.db" && name != "actor.db-wal" && name != "actor.db-shm" {
			t.Errorf("the run left %q beside the database", name)
		}
	}
}

func TestAWALDatabaseInAReadOnlyDirectoryNamesWALAsTheReason(t *testing.T) {
	f := newFixture(t, "")
	f.intoWAL(t)
	dir := filepath.Dir(f.path)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	got := f.run()

	if got.code != 2 {
		t.Fatalf("outcome = %+v, want exit 2", got)
	}
	if !strings.Contains(got.stderr, "WAL mode") {
		t.Fatalf("stderr = %q, want the WAL mode named as the reason", got.stderr)
	}
}

func TestADatabaseNotInWALModeGetsNoWALHint(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)
	sqlDB, _ := f.db.DB()
	sqlDB.Close()

	if inWALMode(f.path) {
		t.Fatalf("a rollback-journal database was taken for WAL")
	}
}
