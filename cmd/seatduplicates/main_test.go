package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

const (
	testInstance = "test"
	password     = "Sup3rSecretPw"
)

type fixture struct {
	path  string
	db    *gorm.DB
	seats string
	actor mwanachamaactor.Actor
	ward  mwanachamaactor.Group
	kind  mwanachamaactor.RoleKind
}

func newFixture(t *testing.T, mount string) *fixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "actor.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecForMount(testInstance, mount)
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	if err := mwanachamaactor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaactor.NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	o, _ := s.ByRole("actor_role_assignment")
	for _, idx := range o.Indexes {
		if idx.Name == "held_once" {
			if err := db.Exec("drop index " + s.IndexFor(o, idx)).Error; err != nil {
				t.Fatalf("drop the held_once index: %v", err)
			}
		}
	}
	ctx := context.Background()
	actor, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Grace Adhiambo"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	ward, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward A"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	f := &fixture{path: path, db: db, seats: s.TableFor(o), actor: actor, ward: ward, kind: kind}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { sqlDB.Close() })
	}
	return f
}

func (f *fixture) seat(t *testing.T, id string, active bool, grantedAt string) {
	t.Helper()
	f.seatIn(t, id, f.ward.ID, active, grantedAt)
}

func (f *fixture) seatIn(t *testing.T, id, groupID string, active bool, grantedAt string) {
	t.Helper()
	err := f.db.Exec(
		"insert into "+f.seats+" (id, actor_id, group_id, kind_id, active, granted_at) values (?, ?, ?, ?, ?, ?)",
		id, f.actor.ID, groupID, f.kind.ID, active, grantedAt,
	).Error
	if err != nil {
		t.Fatalf("insert seat %s: %v", id, err)
	}
}

func (f *fixture) duplicate(t *testing.T) {
	t.Helper()
	f.seat(t, "newer", true, "2026-10-09T09:00:00Z")
	f.seat(t, "older", true, "2026-10-01T09:00:00Z")
}

type outcome struct {
	code   int
	stdout string
	stderr string
}

func runWith(env map[string]string, args ...string) outcome {
	var stdout, stderr bytes.Buffer
	code := run(args, func(name string) string { return env[name] }, &stdout, &stderr)
	return outcome{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func (f *fixture) run(args ...string) outcome {
	return runWith(map[string]string{"SQLITE_PATH": f.path}, append([]string{"-dialect", "sqlite", "-instance", testInstance}, args...)...)
}

func TestExitsZeroAndCountsNothingWhenNoSeatIsHeldTwice(t *testing.T) {
	f := newFixture(t, "")
	f.seat(t, "only", true, "2026-10-01T09:00:00Z")

	got := f.run()

	if got.code != 0 || got.stdout != "duplicates: 0\n" || got.stderr != "" {
		t.Fatalf("outcome = %+v, want exit 0 with only the count", got)
	}
}

func TestExitsZeroOnADatabaseWithNoSeatsAtAll(t *testing.T) {
	f := newFixture(t, "")

	got := f.run()

	if got.code != 0 || got.stdout != "duplicates: 0\n" {
		t.Fatalf("outcome = %+v, want exit 0 with only the count", got)
	}
}

func TestExitsOneAndPrintsOneLineForADuplicateTriple(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)

	got := f.run()

	want := "actor=" + f.actor.ID + " group=" + f.ward.ID + ` group_name="Ward A" kind=` + f.kind.ID +
		` kind_name="Coordinator" seats=2 ids=older,newer oldest=2026-10-01T09:00:00Z newest=2026-10-09T09:00:00Z` + "\n" +
		"duplicates: 1\n"
	if got.code != 1 || got.stdout != want || got.stderr != "" {
		t.Fatalf("outcome = %+v, want exit 1 and\n%s", got, want)
	}
}

func TestExitsOneWhenOneOfTwoTriplesIsDuplicated(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)
	f.seatIn(t, "elsewhere", "other-group", true, "2026-10-02T09:00:00Z")

	got := f.run()

	if got.code != 1 || !strings.HasSuffix(got.stdout, "duplicates: 1\n") || strings.Count(got.stdout, "\n") != 2 {
		t.Fatalf("outcome = %+v, want exit 1 with one line and a count of 1", got)
	}
}

func TestExitsZeroWhenTheOtherSeatOfTheTripleHasEnded(t *testing.T) {
	f := newFixture(t, "")
	f.seat(t, "live", true, "2026-10-09T09:00:00Z")
	f.seat(t, "ended", false, "2026-10-01T09:00:00Z")

	if got := f.run(); got.code != 0 {
		t.Fatalf("outcome = %+v, want exit 0", got)
	}
}

func TestExitsZeroWhenTheTwoSeatsAreInDifferentGroups(t *testing.T) {
	f := newFixture(t, "")
	f.seat(t, "ward-a", true, "2026-10-01T09:00:00Z")
	f.seatIn(t, "ward-b", "ward-b-id", true, "2026-10-02T09:00:00Z")

	if got := f.run(); got.code != 0 {
		t.Fatalf("outcome = %+v, want exit 0", got)
	}
}

func TestReadsTheTablesOfTheMountItIsGiven(t *testing.T) {
	f := newFixture(t, "directory")
	f.duplicate(t)

	if got := f.run("-mount", "directory"); got.code != 1 {
		t.Fatalf("with the mount: outcome = %+v, want exit 1", got)
	}
	if got := f.run(); got.code != 2 {
		t.Fatalf("without the mount: outcome = %+v, want exit 2 because the default mount has no tables here", got)
	}
}

func TestExitsTwoWhenTheInstanceHasNoTables(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)

	got := runWith(map[string]string{"SQLITE_PATH": f.path}, "-dialect", "sqlite", "-instance", "another")

	if got.code != 2 || got.stdout != "" || got.stderr == "" {
		t.Fatalf("outcome = %+v, want exit 2, nothing on stdout and a reason on stderr", got)
	}
}

func TestExitsTwoOnAnInstanceNameTheSpecRefuses(t *testing.T) {
	f := newFixture(t, "")

	if got := f.run(); got.code != 0 {
		t.Fatalf("control run: outcome = %+v, want exit 0", got)
	}
	got := runWith(map[string]string{"SQLITE_PATH": f.path}, "-dialect", "sqlite", "-instance", "Bad Name;")
	if got.code != 2 {
		t.Fatalf("outcome = %+v, want exit 2", got)
	}
}

func TestExitsTwoOnASchemaNameThatIsNotAnIdentifier(t *testing.T) {
	f := newFixture(t, "")

	if got := f.run("-schema", `main"; drop table x; --`); got.code != 2 {
		t.Fatalf("outcome = %+v, want exit 2", got)
	}
}

func TestReadsThroughAnExplicitSchema(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)

	if got := f.run("-schema", "main"); got.code != 1 {
		t.Fatalf("outcome = %+v, want exit 1", got)
	}
	if got := f.run("-schema", "elsewhere"); got.code != 2 {
		t.Fatalf("outcome = %+v, want exit 2 for a schema that holds no tables", got)
	}
}

func TestExitsTwoWhenTheUsageIsWrong(t *testing.T) {
	f := newFixture(t, "")
	env := map[string]string{"SQLITE_PATH": f.path, "POSTGRES_URL": "postgres://u:" + password + "@127.0.0.1:1/db"}

	cases := map[string][]string{
		"no instance":         {"-dialect", "sqlite"},
		"unknown flag":        {"-dialect", "sqlite", "-instance", testInstance, "-url", "x"},
		"unknown dialect":     {"-dialect", "mysql", "-instance", testInstance},
		"a stray argument":    {"-dialect", "sqlite", "-instance", testInstance, "extra"},
		"the url as argument": {"-instance", testInstance, "postgres://u:" + password + "@127.0.0.1:1/db"},
		"no arguments at all": nil,
	}
	for name, args := range cases {
		got := runWith(env, args...)
		if got.code != 2 || got.stdout != "" {
			t.Errorf("%s: outcome = %+v, want exit 2 and nothing on stdout", name, got)
		}
		if strings.Contains(got.stderr, password) {
			t.Errorf("%s: stderr leaks the password: %s", name, got.stderr)
		}
	}
}

func TestExitsTwoWhenTheEnvironmentVariableIsUnset(t *testing.T) {
	for dialect, variable := range map[string]string{"postgres": "POSTGRES_URL", "sqlite": "SQLITE_PATH"} {
		got := runWith(nil, "-dialect", dialect, "-instance", testInstance)
		if got.code != 2 || !strings.Contains(got.stderr, variable+" is not set") {
			t.Errorf("%s: outcome = %+v, want exit 2 naming %s", dialect, got, variable)
		}
	}
}

func TestExitsTwoAndCreatesNothingWhenTheSQLiteFileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.db")

	got := runWith(map[string]string{"SQLITE_PATH": missing}, "-dialect", "sqlite", "-instance", testInstance)

	if got.code != 2 {
		t.Fatalf("outcome = %+v, want exit 2", got)
	}
	if strings.Contains(got.stderr, missing) {
		t.Fatalf("stderr prints the path: %s", got.stderr)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("the directory now holds %d entries, want none", len(entries))
	}
}

func TestNeverPrintsTheSQLitePath(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)

	got := f.run()

	if strings.Contains(got.stdout+got.stderr, f.path) || strings.Contains(got.stdout+got.stderr, filepath.Dir(f.path)) {
		t.Fatalf("output prints the path: %+v", got)
	}
}

func TestNeverPrintsAPostgresPasswordWhenTheConnectionFails(t *testing.T) {
	locations := map[string]string{
		"unreachable uri":       "postgres://wakala_api.abcdefghijklmnopqrst:" + password + "@127.0.0.1:1/postgres?sslmode=disable",
		"unparseable port":      "postgres://wakala_api.abcdefghijklmnopqrst:" + password + "@127.0.0.1:notaport/postgres",
		"unreachable keyword":   "host=127.0.0.1 port=1 user=wakala_api.abcdefghijklmnopqrst password=" + password + " dbname=postgres sslmode=disable",
		"quoted keyword":        "host=127.0.0.1 port=1 user=u password='" + password + "' dbname=postgres sslmode=disable",
		"password in the query": "postgres://u@127.0.0.1:1/postgres?password=" + password + "&sslmode=disable",
		"not a url at all":      "nonsense " + password,
	}
	for name, location := range locations {
		got := runWith(map[string]string{"POSTGRES_URL": location}, "-instance", testInstance)
		if got.code != 2 {
			t.Errorf("%s: exit %d, want 2: %+v", name, got.code, got)
		}
		for _, leak := range []string{password, "wakala_api.abcdefghijklmnopqrst", location} {
			if strings.Contains(got.stdout+got.stderr, leak) {
				t.Errorf("%s: output holds %q:\n%s", name, leak, got.stdout+got.stderr)
			}
		}
	}
}

func TestOpenedReadOnlyTheDatabaseAnswersAndRefusesEveryWrite(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)
	before := fileState(t, f.path)

	db, closeDB, err := openReadOnly("sqlite", f.path)
	if err != nil {
		t.Fatalf("openReadOnly: %v", err)
	}
	defer closeDB()

	s, _ := mwanachamaactor.SpecForMount(testInstance, "")
	found, err := mwanachamaactor.FindSeatDuplicates(context.Background(), db, s, "")
	if err != nil || len(found) != 1 {
		t.Fatalf("FindSeatDuplicates through the read-only connection = %+v, %v, want the one duplicate", found, err)
	}

	writes := map[string]string{
		"insert": "insert into " + f.seats + " (id, actor_id, group_id, kind_id, active) values ('x', 'a', 'g', 'k', true)",
		"update": "update " + f.seats + " set active = false",
		"delete": "delete from " + f.seats,
		"create": "create table scratch (a text)",
		"index":  "create unique index held_once_here on " + f.seats + " (actor_id, group_id, kind_id) where active = true",
		"drop":   "drop table " + f.seats,
	}
	for name, statement := range writes {
		if err := db.Exec(statement).Error; err == nil {
			t.Errorf("%s through the read-only connection succeeded", name)
		}
	}

	closeDB()
	if after := fileState(t, f.path); after != before {
		t.Fatalf("the database file changed: before %+v, after %+v", before, after)
	}
}

func TestStillAnswersWhenTheFileAndItsDirectoryCannotBeWritten(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)
	sqlDB, _ := f.db.DB()
	sqlDB.Close()
	dir := filepath.Dir(f.path)
	if err := os.Chmod(f.path, 0o444); err != nil {
		t.Fatalf("chmod file: %v", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	before := fileState(t, f.path)

	got := f.run()

	if got.code != 1 {
		t.Fatalf("outcome = %+v, want exit 1", got)
	}
	if after := fileState(t, f.path); after != before {
		t.Fatalf("the database file changed: before %+v, after %+v", before, after)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("the directory holds %d entries after the run, want only the database", len(entries))
	}
}

func TestARunLeavesTheDatabaseFileByteForByteTheSame(t *testing.T) {
	f := newFixture(t, "")
	f.duplicate(t)
	sqlDB, _ := f.db.DB()
	sqlDB.Close()
	before := fileState(t, f.path)

	f.run()

	if after := fileState(t, f.path); after != before {
		t.Fatalf("the database file changed: before %+v, after %+v", before, after)
	}
}

type state struct {
	size    int64
	modTime int64
	sum     [sha256.Size]byte
}

func fileState(t *testing.T, path string) state {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return state{size: info.Size(), modTime: info.ModTime().UnixNano(), sum: sha256.Sum256(raw)}
}
