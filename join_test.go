package mwanachamaactor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func seedJoin(t *testing.T, mgr mwanachamaactor.UserManager, mode string) (mwanachamaactor.Group, mwanachamaactor.Actor) {
	t.Helper()
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4", JoinMode: mode})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	return g, a
}

func registered(t *testing.T, mgr mwanachamaactor.UserManager, actorID, groupID string) bool {
	t.Helper()
	held, err := mgr.ListGroupsForActor(context.Background(), actorID)
	if err != nil {
		t.Fatalf("ListGroupsForActor: %v", err)
	}
	for _, a := range held {
		if a.GroupID == groupID {
			return true
		}
	}
	return false
}

func TestANewGroupStartsConfirmedAndShown(t *testing.T) {
	mgr := newTestManager(t)
	g, err := mgr.CreateGroup(context.Background(), mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	got, err := mgr.GetGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.JoinMode != models.JoinConfirmed || got.Hidden {
		t.Errorf("join_mode = %q hidden = %v, want confirmed and shown", got.JoinMode, got.Hidden)
	}
}

func TestAnUnknownJoinModeIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 4", JoinMode: "anyone"}); !errors.Is(err, mwanachamaactor.ErrInvalidJoinMode) {
		t.Errorf("CreateGroup err = %v, want ErrInvalidJoinMode", err)
	}
	g, _ := seedJoin(t, mgr, "")
	if _, err := mgr.SetGroupJoinMode(ctx, g.ID, "anyone"); !errors.Is(err, mwanachamaactor.ErrInvalidJoinMode) {
		t.Errorf("SetGroupJoinMode err = %v, want ErrInvalidJoinMode", err)
	}
	if _, err := mgr.SetGroupJoinMode(ctx, "missing", models.JoinOpen); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Errorf("SetGroupJoinMode on a missing group err = %v, want ErrGroupNotFound", err)
	}
}

func TestJoiningAnOpenGroupRegistersAtOnce(t *testing.T) {
	mgr := newTestManager(t)
	g, a := seedJoin(t, mgr, models.JoinOpen)

	out, err := mgr.JoinGroup(context.Background(), g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if out.Status != models.JoinedNow || out.Assignment == nil || out.Request != nil {
		t.Fatalf("outcome = %+v, want joined with an assignment and no request", out)
	}
	if !registered(t, mgr, a.ID, g.ID) {
		t.Error("the actor is not registered after joining an open group")
	}

	again, err := mgr.JoinGroup(context.Background(), g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup again: %v", err)
	}
	if again.Status != models.AlreadyAMember {
		t.Errorf("status = %q, want member", again.Status)
	}
}

func TestARequestToAConfirmedGroupGrantsNothingUntilApproved(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)

	out, err := mgr.JoinGroup(ctx, g.ID, a.ID, "  I live on this street.  ")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if out.Status != models.JoinRequested || out.Request == nil {
		t.Fatalf("outcome = %+v, want requested", out)
	}
	req := *out.Request
	if req.State != models.JoinRequestPending || req.Appeal || req.Message != "I live on this street." || req.RequestedAt == "" {
		t.Errorf("request = %+v", req)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Fatal("a pending request registered the actor")
	}

	twice, err := mgr.JoinGroup(ctx, g.ID, a.ID, "Still waiting.")
	if err != nil {
		t.Fatalf("JoinGroup twice: %v", err)
	}
	if twice.Request.ID != req.ID || twice.Request.Appeal || twice.Request.Message != "Still waiting." {
		t.Errorf("asking twice = %+v, want the same request, not an appeal, with the new message", twice.Request)
	}
	pending, err := mgr.ListJoinRequestsForGroup(ctx, g.ID, true)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %v, %v; want exactly one", pending, err)
	}

	seatLeader(t, mgr, g.ID, "leader-1")
	seatLeader(t, mgr, g.ID, "leader-2")
	approved, err := mgr.ApproveJoinRequest(ctx, req.ID, "leader-1")
	if err != nil {
		t.Fatalf("ApproveJoinRequest: %v", err)
	}
	if approved.State != models.JoinRequestApproved || approved.DecidedBy != "leader-1" || approved.DecidedAt == "" {
		t.Errorf("approved = %+v", approved)
	}
	if !registered(t, mgr, a.ID, g.ID) {
		t.Error("approval did not register the actor")
	}
	if _, err := mgr.ApproveJoinRequest(ctx, req.ID, "leader-2"); !errors.Is(err, mwanachamaactor.ErrJoinRequestDecided) {
		t.Errorf("second approval err = %v, want ErrJoinRequestDecided", err)
	}
	if left, _ := mgr.ListJoinRequestsForGroup(ctx, g.ID, true); len(left) != 0 {
		t.Errorf("pending after approval = %v, want none", left)
	}
}

func TestADeclinedActorMayAskAgainAsAnAppeal(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)

	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "Please.")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	seatLeader(t, mgr, g.ID, "leader-1")
	declined, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", true)
	if err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	if declined.State != models.JoinRequestDeclined || !declined.Silent || declined.DecidedBy != "leader-1" {
		t.Errorf("declined = %+v, want a silent decline by leader-1", declined)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Fatal("a decline registered the actor")
	}

	appeal, err := mgr.JoinGroup(ctx, g.ID, a.ID, "I have moved here since.")
	if err != nil {
		t.Fatalf("JoinGroup appeal: %v", err)
	}
	r := appeal.Request
	if r.ID != first.Request.ID {
		t.Errorf("appeal made request %s, want the original %s updated", r.ID, first.Request.ID)
	}
	if r.State != models.JoinRequestPending || !r.Appeal || r.Silent || r.DecidedBy != "" || r.DecidedAt != "" {
		t.Errorf("appeal = %+v, want pending, flagged, with the decision cleared", r)
	}
	if r.Message != "I have moved here since." || r.RequestedAt != first.Request.RequestedAt {
		t.Errorf("appeal = %+v, want the new message and the first ask's time", r)
	}
	stored, err := mgr.GetJoinRequest(ctx, r.ID)
	if err != nil || !stored.Appeal || stored.State != models.JoinRequestPending {
		t.Errorf("stored = %+v, %v", stored, err)
	}

	mine, err := mgr.ListJoinRequestsForActor(ctx, a.ID)
	if err != nil || len(mine) != 1 {
		t.Errorf("the actor's requests = %v, %v; want one", mine, err)
	}
}

func TestADecisionNamesWhoMadeIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	out, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.ApproveJoinRequest(ctx, out.Request.ID, " "); !errors.Is(err, mwanachamaactor.ErrInvalidJoinRequest) {
		t.Errorf("approve with no decider err = %v, want ErrInvalidJoinRequest", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, "missing", "leader-1", false); !errors.Is(err, mwanachamaactor.ErrJoinRequestNotFound) {
		t.Errorf("decline of a missing request err = %v, want ErrJoinRequestNotFound", err)
	}
}

func TestAJoinMessageIsAtMostFiveHundredCharacters(t *testing.T) {
	mgr := newTestManager(t)
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	long := strings.Repeat("é", models.MaxJoinMessageLength+1)
	if _, err := mgr.JoinGroup(context.Background(), g.ID, a.ID, long); !errors.Is(err, mwanachamaactor.ErrJoinMessageTooLong) {
		t.Errorf("err = %v, want ErrJoinMessageTooLong", err)
	}
}

func TestJoiningNeedsAnActorThatExists(t *testing.T) {
	mgr := newTestManager(t)
	g, _ := seedJoin(t, mgr, models.JoinOpen)
	if _, err := mgr.JoinGroup(context.Background(), g.ID, "nobody", ""); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Errorf("err = %v, want ErrActorNotFound", err)
	}
}

func TestAHiddenGroupCannotBeJoinedOrListed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinOpen)
	hidden, err := mgr.SetGroupHidden(ctx, g.ID, true)
	if err != nil || !hidden.Hidden {
		t.Fatalf("SetGroupHidden = %+v, %v", hidden, err)
	}
	if _, err := mgr.JoinGroup(ctx, g.ID, a.ID, ""); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Errorf("join of a hidden group err = %v, want ErrGroupNotFound", err)
	}
	joinable, err := mgr.ListJoinableGroups(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	if len(joinable) != 0 {
		t.Errorf("joinable = %v, want the hidden group left out", joinable)
	}
}

func TestTheJoinableListLeavesOutWhereTheActorIsRegistered(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	in, a := seedJoin(t, mgr, models.JoinOpen)
	asked, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Choir"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	hidden, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Elders", Hidden: true})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.JoinGroup(ctx, in.ID, a.ID, ""); err != nil {
		t.Fatalf("JoinGroup open: %v", err)
	}
	if _, err := mgr.JoinGroup(ctx, asked.ID, a.ID, ""); err != nil {
		t.Fatalf("JoinGroup confirmed: %v", err)
	}

	joinable, err := mgr.ListJoinableGroups(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	if len(joinable) != 1 || joinable[0].ID != asked.ID {
		t.Errorf("joinable = %+v, want only %s (registered at %s, %s hidden)", joinable, asked.ID, in.ID, hidden.ID)
	}

	someoneElse, err := mgr.ListJoinableGroups(ctx, "another-actor")
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	if len(someoneElse) != 2 {
		t.Errorf("another actor's joinable = %d groups, want 2", len(someoneElse))
	}
}

func TestOpeningAGroupLetsAPendingAskerStraightIn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	asked, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.SetGroupJoinMode(ctx, g.ID, models.JoinOpen); err != nil {
		t.Fatalf("SetGroupJoinMode: %v", err)
	}
	out, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil || out.Status != models.JoinedNow {
		t.Fatalf("JoinGroup = %+v, %v; want joined", out, err)
	}
	req, err := mgr.GetJoinRequest(ctx, asked.Request.ID)
	if err != nil {
		t.Fatalf("GetJoinRequest: %v", err)
	}
	if req.State != models.JoinRequestApproved || req.DecidedBy != "" {
		t.Errorf("request = %+v, want approved by the open policy itself", req)
	}
}

func TestProvisionAddsTheJoinColumnsToAnExistingGroupTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	g, err := mgr.CreateGroup(context.Background(), mwanachamaactor.Group{Name: "Ward 4"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	o, _ := s.ByRole("group")
	table := s.TableFor(o)
	for _, column := range []string{"join_mode", "hidden"} {
		if err := db.Exec("alter table " + table + " drop column " + column).Error; err != nil {
			t.Fatalf("drop %s: %v", column, err)
		}
	}

	again := newTestManagerOn(t, db)
	got, err := again.GetGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.JoinMode != models.JoinConfirmed || got.Hidden {
		t.Errorf("an existing group reads join_mode = %q hidden = %v, want confirmed and shown", got.JoinMode, got.Hidden)
	}
}
