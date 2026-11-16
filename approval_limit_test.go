package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	actor "github.com/aosanya/mwanachama-backend-actor"
)

func limitOf(n int) *int { return &n }

func TestWithinLimit(t *testing.T) {
	cases := []struct {
		name   string
		seat   actor.ActorRoleAssignment
		amount int
		want   bool
	}{
		{"no limit approves any amount", actor.ActorRoleAssignment{Active: true}, 9_000_000, true},
		{"amount under the limit", actor.ActorRoleAssignment{Active: true, ApprovalLimit: limitOf(500_000)}, 499_999, true},
		{"amount equal to the limit", actor.ActorRoleAssignment{Active: true, ApprovalLimit: limitOf(500_000)}, 500_000, true},
		{"amount over the limit", actor.ActorRoleAssignment{Active: true, ApprovalLimit: limitOf(500_000)}, 500_001, false},
		{"a zero limit approves only zero", actor.ActorRoleAssignment{Active: true, ApprovalLimit: limitOf(0)}, 1, false},
		{"a zero limit approves zero", actor.ActorRoleAssignment{Active: true, ApprovalLimit: limitOf(0)}, 0, true},
		{"an ended seat approves nothing", actor.ActorRoleAssignment{Active: false}, 1, false},
		{"a negative amount is refused", actor.ActorRoleAssignment{Active: true}, -1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := actor.WithinLimit(c.seat, c.amount); got != c.want {
				t.Fatalf("WithinLimit = %v, want %v", got, c.want)
			}
		})
	}
}

func TestCheckLimitNamesTheRefusal(t *testing.T) {
	seat := actor.ActorRoleAssignment{Active: true, ApprovalLimit: limitOf(10)}
	if err := actor.CheckLimit(seat, 11); !errors.Is(err, actor.ErrOverApprovalLimit) {
		t.Fatalf("CheckLimit over = %v, want ErrOverApprovalLimit", err)
	}
	if err := actor.CheckLimit(seat, 10); err != nil {
		t.Fatalf("CheckLimit at the limit = %v", err)
	}
}

func newManager(t *testing.T) actor.UserManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := actor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	um, err := actor.NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return um
}

func seatFixture(t *testing.T, um actor.UserManager) (actorID, groupID, kindID string) {
	t.Helper()
	ctx := context.Background()
	a, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Kovács Anna"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	g, err := um.CreateGroup(ctx, actor.Group{Name: "Városi tanács"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	k, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: "Pénzügyi vezető"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	return a.ID, g.ID, k.ID
}

func TestGrantRoleKeepsTheApprovalLimit(t *testing.T) {
	um := newManager(t)
	actorID, groupID, kindID := seatFixture(t, um)
	ctx := context.Background()

	granted, err := um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: kindID, ApprovalLimit: limitOf(250_000)})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	read, err := um.GetRoleAssignment(ctx, granted.ID)
	if err != nil {
		t.Fatalf("GetRoleAssignment: %v", err)
	}
	if read.ApprovalLimit == nil || *read.ApprovalLimit != 250_000 {
		t.Fatalf("stored limit = %v, want 250000", read.ApprovalLimit)
	}
	if actor.WithinLimit(read, 250_001) || !actor.WithinLimit(read, 250_000) {
		t.Fatal("the stored seat does not stop at its limit")
	}
}

func TestGrantRoleWithoutALimitStaysUnlimited(t *testing.T) {
	um := newManager(t)
	actorID, groupID, kindID := seatFixture(t, um)
	ctx := context.Background()

	granted, err := um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: kindID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	read, err := um.GetRoleAssignment(ctx, granted.ID)
	if err != nil {
		t.Fatalf("GetRoleAssignment: %v", err)
	}
	if read.ApprovalLimit != nil {
		t.Fatalf("stored limit = %d, want none", *read.ApprovalLimit)
	}
	if !actor.WithinLimit(read, 1_000_000_000) {
		t.Fatal("a seat granted without a limit stopped an approval")
	}
}

func TestGrantRoleRefusesANegativeLimit(t *testing.T) {
	um := newManager(t)
	actorID, groupID, kindID := seatFixture(t, um)

	_, err := um.GrantRole(context.Background(), actor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: kindID, ApprovalLimit: limitOf(-1)})
	if !errors.Is(err, actor.ErrApprovalLimitNegative) {
		t.Fatalf("GrantRole = %v, want ErrApprovalLimitNegative", err)
	}
}

func TestProvisionAddsTheLimitColumnToAnOlderTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := actor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	object, _ := s.Object("actor_role_assignment")
	table := s.TableFor(object)
	if err := db.Exec("alter table " + table + " drop column approval_limit").Error; err != nil {
		t.Fatalf("drop column: %v", err)
	}
	if err := actor.Provision(db, s); err != nil {
		t.Fatalf("Provision again: %v", err)
	}
	if !db.Migrator().HasColumn(table, "approval_limit") {
		t.Fatal("approval_limit was not added to the older table")
	}
}
