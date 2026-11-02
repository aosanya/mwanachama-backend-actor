package mwanachamaactor_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type seatDuplicateFixture struct {
	db    *gorm.DB
	spec  *spec.Spec
	mgr   mwanachamaactor.UserManager
	seats string
	actor mwanachamaactor.Actor
	ward  mwanachamaactor.Group
	coord mwanachamaactor.RoleKind
}

func newSeatDuplicateFixture(t *testing.T) *seatDuplicateFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "actor.db")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	o, _ := s.ByRole("actor_role_assignment")
	for _, idx := range o.Indexes {
		if idx.Name == "held_once" {
			if err := db.Exec("drop index " + s.IndexFor(o, idx)).Error; err != nil {
				t.Fatalf("drop the held_once index: %v", err)
			}
		}
	}
	kind, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	return &seatDuplicateFixture{
		db:    db,
		spec:  s,
		mgr:   mgr,
		seats: s.TableFor(o),
		actor: mustActor(t, mgr, "Grace Adhiambo"),
		ward:  mustGroup(t, mgr, "Ward A"),
		coord: kind,
	}
}

func (f *seatDuplicateFixture) seat(t *testing.T, id, actorID, groupID, kindID string, active bool, grantedAt string) {
	t.Helper()
	err := f.db.Exec(
		"insert into "+f.seats+" (id, actor_id, group_id, kind_id, active, granted_at) values (?, ?, ?, ?, ?, ?)",
		id, actorID, groupID, kindID, active, grantedAt,
	).Error
	if err != nil {
		t.Fatalf("insert seat %s: %v", id, err)
	}
}

func (f *seatDuplicateFixture) find(t *testing.T) []mwanachamaactor.SeatDuplicate {
	t.Helper()
	found, err := mwanachamaactor.FindSeatDuplicates(context.Background(), f.db, f.spec, "")
	if err != nil {
		t.Fatalf("FindSeatDuplicates: %v", err)
	}
	return found
}

func TestSeatDuplicatesIsEmptyWhenEveryLiveSeatIsAlone(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	f.seat(t, "only", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")

	if found := f.find(t); len(found) != 0 {
		t.Fatalf("duplicates = %+v, want none", found)
	}
}

func TestSeatDuplicatesIsEmptyOnAnEmptySeatTable(t *testing.T) {
	f := newSeatDuplicateFixture(t)

	if found := f.find(t); len(found) != 0 {
		t.Fatalf("duplicates = %+v, want none", found)
	}
}

func TestSeatDuplicatesNamesOneTripleHeldTwiceAndLeavesTheRestOut(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	other := mustActor(t, f.mgr, "Baraka Otieno")
	f.seat(t, "newer", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-09T09:00:00Z")
	f.seat(t, "older", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "alone", other.ID, f.ward.ID, f.coord.ID, true, "2026-10-02T09:00:00Z")

	found := f.find(t)

	want := []mwanachamaactor.SeatDuplicate{{
		ActorID:         f.actor.ID,
		GroupID:         f.ward.ID,
		GroupName:       "Ward A",
		KindID:          f.coord.ID,
		KindName:        "Coordinator",
		Seats:           2,
		SeatIDs:         []string{"older", "newer"},
		OldestGrantedAt: "2026-10-01T09:00:00Z",
		NewestGrantedAt: "2026-10-09T09:00:00Z",
	}}
	if !reflect.DeepEqual(found, want) {
		t.Fatalf("duplicates = %+v, want %+v", found, want)
	}
}

func TestSeatDuplicatesCountsAllSeatsOfATripleHeldThreeTimes(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	f.seat(t, "a", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-03T09:00:00Z")
	f.seat(t, "b", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "c", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-02T09:00:00Z")

	found := f.find(t)

	if len(found) != 1 || found[0].Seats != 3 || !reflect.DeepEqual(found[0].SeatIDs, []string{"b", "c", "a"}) {
		t.Fatalf("duplicates = %+v, want one triple of 3 seats listed oldest first", found)
	}
}

func TestSeatDuplicatesReportsEachTripleOnItsOwn(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	other := mustActor(t, f.mgr, "Baraka Otieno")
	f.seat(t, "g1", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "g2", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-02T09:00:00Z")
	f.seat(t, "b1", other.ID, f.ward.ID, f.coord.ID, true, "2026-10-03T09:00:00Z")
	f.seat(t, "b2", other.ID, f.ward.ID, f.coord.ID, true, "2026-10-04T09:00:00Z")

	found := f.find(t)

	if len(found) != 2 {
		t.Fatalf("duplicates = %+v, want two triples", found)
	}
	if found[0].ActorID == found[1].ActorID {
		t.Fatalf("duplicates = %+v, want one per actor", found)
	}
}

func TestSeatDuplicatesIgnoresAnEndedSeatBesideALiveOne(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	f.seat(t, "live", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-09T09:00:00Z")
	f.seat(t, "ended", f.actor.ID, f.ward.ID, f.coord.ID, false, "2026-10-01T09:00:00Z")
	f.seat(t, "ended-too", f.actor.ID, f.ward.ID, f.coord.ID, false, "2026-10-02T09:00:00Z")

	if found := f.find(t); len(found) != 0 {
		t.Fatalf("duplicates = %+v, want none: only one of the three seats is live", found)
	}
}

func TestSeatDuplicatesIgnoresTwoGroupsOfTheSameKind(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	ward2 := mustGroup(t, f.mgr, "Ward B")
	f.seat(t, "ward-a", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "ward-b", f.actor.ID, ward2.ID, f.coord.ID, true, "2026-10-02T09:00:00Z")

	if found := f.find(t); len(found) != 0 {
		t.Fatalf("duplicates = %+v, want none: the two seats are in different groups", found)
	}
}

func TestSeatDuplicatesIgnoresTwoKindsAtTheSameGroup(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	treasurer, err := f.mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	f.seat(t, "coordinator", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "treasurer", f.actor.ID, f.ward.ID, treasurer.ID, true, "2026-10-02T09:00:00Z")

	if found := f.find(t); len(found) != 0 {
		t.Fatalf("duplicates = %+v, want none: the two seats are of different kinds", found)
	}
}

func TestSeatDuplicatesStillReportsATripleWhoseGroupAndKindRowsAreGone(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	f.seat(t, "one", f.actor.ID, "no-such-group", "no-such-kind", true, "2026-10-01T09:00:00Z")
	f.seat(t, "two", f.actor.ID, "no-such-group", "no-such-kind", true, "2026-10-02T09:00:00Z")

	found := f.find(t)

	if len(found) != 1 || found[0].GroupName != "" || found[0].KindName != "" || found[0].Seats != 2 {
		t.Fatalf("duplicates = %+v, want one triple of 2 seats with empty names", found)
	}
}

func TestSeatDuplicatesReadsTheSameThroughAnExplicitSchema(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	f.seat(t, "one", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "two", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-02T09:00:00Z")

	found, err := mwanachamaactor.FindSeatDuplicates(context.Background(), f.db, f.spec, "main")
	if err != nil {
		t.Fatalf("FindSeatDuplicates in schema main: %v", err)
	}
	if len(found) != 1 || found[0].Seats != 2 {
		t.Fatalf("duplicates = %+v, want the one triple", found)
	}
}

func TestSeatDuplicatesFailsOnASchemaThatHoldsNoTables(t *testing.T) {
	f := newSeatDuplicateFixture(t)

	_, err := mwanachamaactor.FindSeatDuplicates(context.Background(), f.db, f.spec, "elsewhere")
	if err == nil || !strings.Contains(err.Error(), f.seats) {
		t.Fatalf("error = %v, want one that names the seat table %s", err, f.seats)
	}
}

func TestSeatDuplicatesRefusesASchemaNameThatIsNotAnIdentifier(t *testing.T) {
	f := newSeatDuplicateFixture(t)

	for _, schema := range []string{`a"b`, "a b", "a;drop table x", `"quoted"`, "1st", "a.b"} {
		_, err := mwanachamaactor.FindSeatDuplicates(context.Background(), f.db, f.spec, schema)
		if !errors.Is(err, mwanachamaactor.ErrSchemaName) {
			t.Errorf("schema %q: error = %v, want ErrSchemaName", schema, err)
		}
	}
}

func TestSeatDuplicatesFailsWhenTheSeatTableIsMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "empty.db")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}

	if _, err := mwanachamaactor.FindSeatDuplicates(context.Background(), db, s, ""); err == nil {
		t.Fatal("a database without the seat table was read without error")
	}
}

func TestSeatDuplicatesWritesNothing(t *testing.T) {
	f := newSeatDuplicateFixture(t)
	f.seat(t, "one", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-01T09:00:00Z")
	f.seat(t, "two", f.actor.ID, f.ward.ID, f.coord.ID, true, "2026-10-02T09:00:00Z")
	before := snapshotOf(t, f.db)

	f.find(t)

	if after := snapshotOf(t, f.db); !reflect.DeepEqual(before, after) {
		t.Fatalf("the database changed:\nbefore %v\nafter  %v", before, after)
	}
}

func snapshotOf(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()
	var tables []string
	if err := db.Raw("select name from sqlite_master where type = 'table' and name not like 'sqlite_%'").Scan(&tables).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var indexes []string
	if err := db.Raw("select name from sqlite_master where type = 'index'").Scan(&indexes).Error; err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	out := map[string]int64{"indexes": int64(len(indexes))}
	for _, table := range tables {
		var rows int64
		if err := db.Table(table).Count(&rows).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		out[table] = rows
	}
	return out
}
