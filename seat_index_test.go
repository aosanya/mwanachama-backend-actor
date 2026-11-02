package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

type seatIndexFixture struct {
	mgr   mwanachamaactor.UserManager
	db    *gorm.DB
	table string
	seat  mwanachamaactor.ActorRoleAssignment
}

func newSeatIndexFixture(t *testing.T) seatIndexFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	o, _ := s.ByRole("actor_role_assignment")
	grace := mustActor(t, mgr, "Grace Adhiambo")
	ward := mustGroup(t, mgr, "Ward A")
	kind, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	seat, err := mgr.GrantRole(context.Background(), mwanachamaactor.ActorRoleAssignment{ActorID: grace.ID, GroupID: ward.ID, KindID: kind.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return seatIndexFixture{mgr: mgr, db: db, table: s.TableFor(o), seat: seat}
}

func (f seatIndexFixture) insertSeat(id string, active bool) error {
	return f.db.Exec(
		"insert into "+f.table+" (id, actor_id, group_id, kind_id, active) values (?, ?, ?, ?, ?)",
		id, f.seat.ActorID, f.seat.GroupID, f.seat.KindID, active,
	).Error
}

func TestTheDatabaseRefusesASecondLiveSeatOfTheSameKindAtTheSameGroup(t *testing.T) {
	f := newSeatIndexFixture(t)

	if err := f.insertSeat("second-live", true); err == nil {
		t.Fatal("a second live seat of the same actor, group and kind was inserted, want the unique index to refuse it")
	}
	if err := f.insertSeat("an-ended-one", false); err != nil {
		t.Fatalf("an ended seat of the same triple was refused: %v", err)
	}
	if err := f.insertSeat("another-ended-one", false); err != nil {
		t.Fatalf("a second ended seat of the same triple was refused: %v", err)
	}
}

func TestAGrantThatLosesTheRaceAtTheIndexIsSeatAlreadyHeld(t *testing.T) {
	f := newSeatIndexFixture(t)
	if err := f.mgr.RevokeRole(context.Background(), f.seat.ID); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	slipped := false
	err := f.db.Callback().Create().Before("gorm:create").Register("slip_in_the_winner", func(tx *gorm.DB) {
		if slipped || tx.Statement.Table != f.table {
			return
		}
		slipped = true
		if err := tx.Session(&gorm.Session{NewDB: true}).Exec(
			"insert into "+f.table+" (id, actor_id, group_id, kind_id, active) values (?, ?, ?, ?, ?)",
			"the-winner", f.seat.ActorID, f.seat.GroupID, f.seat.KindID, true,
		).Error; err != nil {
			t.Errorf("slip in the winning grant: %v", err)
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, err = f.mgr.GrantRole(context.Background(), mwanachamaactor.ActorRoleAssignment{ActorID: f.seat.ActorID, GroupID: f.seat.GroupID, KindID: f.seat.KindID})
	if !errors.Is(err, mwanachamaactor.ErrSeatAlreadyHeld) {
		t.Fatalf("the grant that lost the race = %v, want ErrSeatAlreadyHeld", err)
	}
	live, _ := f.mgr.ListRoleAssignmentsForActor(context.Background(), f.seat.ActorID, true)
	if len(live) != 0 {
		t.Fatalf("live seats after the losing grant rolled back = %+v, want none of its own", live)
	}
}
