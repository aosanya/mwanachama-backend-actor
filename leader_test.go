package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func leadingKind(t *testing.T, mgr mwanachamaactor.UserManager) mwanachamaactor.RoleKind {
	t.Helper()
	k, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Chair", LeadsGroup: true})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	return k
}

func seatLeader(t *testing.T, mgr mwanachamaactor.UserManager, groupID, actorID string) mwanachamaactor.ActorRoleAssignment {
	t.Helper()
	seat, err := mgr.GrantRole(context.Background(), mwanachamaactor.ActorRoleAssignment{
		ActorID: actorID, GroupID: groupID, KindID: leadingKind(t, mgr).ID,
	})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return seat
}

func leads(t *testing.T, mgr mwanachamaactor.UserManager, groupID, actorID string) bool {
	t.Helper()
	got, err := mgr.IsGroupLeader(context.Background(), groupID, actorID)
	if err != nil {
		t.Fatalf("IsGroupLeader: %v", err)
	}
	return got
}

func TestARoleKindLeadsNothingUntilTheAgencySaysSo(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	treasurer, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if treasurer.LeadsGroup {
		t.Fatal("a new kind leads its group by default")
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: a.ID, GroupID: g.ID, KindID: treasurer.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if leads(t, mgr, g.ID, a.ID) {
		t.Fatal("a seat of a kind that does not lead made its holder a leader")
	}

	marked, err := mgr.SetRoleKindLeadsGroup(ctx, treasurer.ID, true)
	if err != nil || !marked.LeadsGroup {
		t.Fatalf("SetRoleKindLeadsGroup = %+v, %v; want leading", marked, err)
	}
	if read, _ := mgr.GetRoleKind(ctx, treasurer.ID); !read.LeadsGroup {
		t.Error("the mark did not persist")
	}
	if !leads(t, mgr, g.ID, a.ID) {
		t.Fatal("marking the kind did not make its holder a leader")
	}

	if _, err := mgr.SetRoleKindLeadsGroup(ctx, treasurer.ID, false); err != nil {
		t.Fatalf("SetRoleKindLeadsGroup off: %v", err)
	}
	if leads(t, mgr, g.ID, a.ID) {
		t.Error("clearing the mark left the holder a leader")
	}
	if _, err := mgr.SetRoleKindLeadsGroup(ctx, "missing", true); !errors.Is(err, mwanachamaactor.ErrRoleKindNotFound) {
		t.Errorf("a missing kind err = %v, want ErrRoleKindNotFound", err)
	}
}

func TestALeaderLeadsOnlyTheGroupTheirSeatIsIn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	other, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 5"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	seat := seatLeader(t, mgr, g.ID, a.ID)

	if !leads(t, mgr, g.ID, a.ID) || leads(t, mgr, other.ID, a.ID) {
		t.Fatal("a leading seat did not lead exactly its own group")
	}
	led, err := mgr.ListLedGroups(ctx, a.ID)
	if err != nil || len(led) != 1 || led[0].ID != g.ID {
		t.Fatalf("ListLedGroups = %v, %v; want only %s", led, err, g.ID)
	}

	if err := mgr.StepDownRole(ctx, seat.ID); err != nil {
		t.Fatalf("StepDownRole: %v", err)
	}
	if leads(t, mgr, g.ID, a.ID) {
		t.Error("an ended seat still leads")
	}
	led, err = mgr.ListLedGroups(ctx, a.ID)
	if err != nil || len(led) != 0 {
		t.Errorf("ListLedGroups after stepping down = %v, %v; want none", led, err)
	}
}

func TestOnlyALeaderDecidesAJoinRequest(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	other, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 5"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	seatLeader(t, mgr, other.ID, "leader-elsewhere")
	out, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}

	for _, decider := range []string{"stranger", "leader-elsewhere", a.ID} {
		if _, err := mgr.ApproveJoinRequest(ctx, out.Request.ID, decider); !errors.Is(err, mwanachamaactor.ErrNotGroupLeader) {
			t.Errorf("%s approving err = %v, want ErrNotGroupLeader", decider, err)
		}
		if _, err := mgr.DeclineJoinRequest(ctx, out.Request.ID, decider, false); !errors.Is(err, mwanachamaactor.ErrNotGroupLeader) {
			t.Errorf("%s declining err = %v, want ErrNotGroupLeader", decider, err)
		}
	}
	if req, _ := mgr.GetJoinRequest(ctx, out.Request.ID); req.State != models.JoinRequestPending {
		t.Fatalf("a refused decision moved the request to %q", req.State)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Fatal("a refused approval registered the actor")
	}

	seatLeader(t, mgr, g.ID, "chair")
	if _, err := mgr.ApproveJoinRequest(ctx, out.Request.ID, "chair"); err != nil {
		t.Fatalf("the group's leader approving: %v", err)
	}
}

func TestOnlyALeaderInvites(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "stranger", ""); !errors.Is(err, mwanachamaactor.ErrNotGroupLeader) {
		t.Fatalf("a non-leader inviting err = %v, want ErrNotGroupLeader", err)
	}
	if _, err := mgr.InviteToGroup(ctx, g.ID, "missing", "stranger", ""); !errors.Is(err, mwanachamaactor.ErrNotGroupLeader) {
		t.Errorf("a non-leader inviting a missing actor err = %v, want ErrNotGroupLeader, so actors cannot be probed", err)
	}
	invitations, err := mgr.ListInvitationsForActor(ctx, a.ID)
	if err != nil || len(invitations) != 0 {
		t.Fatalf("invitations = %v, %v; want none written", invitations, err)
	}
	seatLeader(t, mgr, g.ID, "chair")
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "chair", ""); err != nil {
		t.Fatalf("the group's leader inviting: %v", err)
	}
}

func TestProvisionAddsLeadsGroupToAnExistingRoleKindTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	k, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Chair"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	o, _ := s.ByRole("role_kind")
	if err := db.Exec("alter table " + s.TableFor(o) + " drop column leads_group").Error; err != nil {
		t.Fatalf("drop leads_group: %v", err)
	}

	again := newTestManagerOn(t, db)
	got, err := again.GetRoleKind(context.Background(), k.ID)
	if err != nil {
		t.Fatalf("GetRoleKind: %v", err)
	}
	if got.LeadsGroup {
		t.Error("an existing kind reads as leading its group")
	}
	if _, err := again.SetRoleKindLeadsGroup(context.Background(), k.ID, true); err != nil {
		t.Errorf("SetRoleKindLeadsGroup on the restored column: %v", err)
	}
}
