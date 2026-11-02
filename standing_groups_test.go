package mwanachamaactor_test

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestStandingNamesTheGroupsOfAPersonWhoseSeatsPassSqlitesVariableLimit(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)
	ctx := context.Background()
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	seatObject, _ := s.ByRole("actor_role_assignment")
	person := mustActor(t, mgr, "Wide Ward")
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Member"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	named := mustGroup(t, mgr, "Named Ward")
	const seats = 33000
	err = db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < seats; i++ {
			groupID := fmt.Sprintf("no-such-group-%d", i)
			if i == 0 {
				groupID = named.ID
			}
			if err := tx.Exec(
				"insert into "+s.TableFor(seatObject)+" (id, actor_id, group_id, kind_id, active, granted_at) values (?, ?, ?, ?, ?, ?)",
				fmt.Sprintf("wide-seat-%d", i), person.ID, groupID, kind.ID, true, "2999-01-01T00:00:00Z",
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("insert seats: %v", err)
	}
	standing, err := mgr.GetActorStanding(ctx, person.ID)
	if err != nil {
		t.Fatalf("GetActorStanding with %d seats: %v", seats, err)
	}
	if len(standing.Seats) != seats {
		t.Fatalf("standing lists %d seats; want %d", len(standing.Seats), seats)
	}
	if got := standingSeat(t, standing.Seats, "wide-seat-0").GroupName; got != "Named Ward" {
		t.Fatalf("the first seat's group_name = %q; want Named Ward", got)
	}
}

func TestStandingKeepsTheGranterIdOfASoftDeletedGranterAndLeavesItsNameOut(t *testing.T) {
	mgr, db := newTestManagerWithDB(t)
	ctx := context.Background()
	person := mustActor(t, mgr, "Seated Person")
	granter := mustActor(t, mgr, "Gone Granter")
	ward := mustGroup(t, mgr, "Granted Ward")
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Member"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	seat, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: person.ID, GroupID: ward.ID, KindID: kind.ID, GrantedBy: granter.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	softDelete(t, db, "actor", granter.ID)
	standing, err := mgr.GetActorStanding(ctx, person.ID)
	if err != nil {
		t.Fatalf("GetActorStanding: %v", err)
	}
	got := standingSeat(t, standing.Seats, seat.ID)
	if got.GrantedBy != granter.ID || got.GrantedByName != "" {
		t.Fatalf("seat granted by a deleted actor = granted_by %q, granted_by_name %q; want the id %q and no name", got.GrantedBy, got.GrantedByName, granter.ID)
	}
}
