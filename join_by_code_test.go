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

func hiddenGroup(t *testing.T, mgr mwanachamaactor.UserManager, mode string) (mwanachamaactor.Group, mwanachamaactor.Actor) {
	t.Helper()
	g, a := seedJoin(t, mgr, mode)
	if _, err := mgr.SetGroupHidden(context.Background(), g.ID, true); err != nil {
		t.Fatalf("SetGroupHidden: %v", err)
	}
	return g, a
}

func TestACodeFindsAHiddenGroupWithItsJoinModeAndTheCallersState(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := hiddenGroup(t, mgr, models.JoinConfirmed)
	if !strings.HasPrefix(g.Code, "G-") {
		t.Fatalf("group code = %q, want G-<n>", g.Code)
	}

	found, err := mgr.FindGroupByCode(ctx, "  "+strings.ToLower(g.Code)+" ", a.ID)
	if err != nil {
		t.Fatalf("FindGroupByCode: %v", err)
	}
	if found.Group.ID != g.ID || !found.Group.Hidden || found.JoinMode != models.JoinConfirmed {
		t.Errorf("found = %+v, want the hidden confirmed group", found)
	}
	if found.State != models.CallerIsOutside || found.Request != nil {
		t.Errorf("state = %q request = %v, want none", found.State, found.Request)
	}

	if _, err := mgr.JoinGroupByCode(ctx, g.Code, a.ID, "I live here."); err != nil {
		t.Fatalf("JoinGroupByCode: %v", err)
	}
	found, err = mgr.FindGroupByCode(ctx, g.Code, a.ID)
	if err != nil || found.State != models.CallerRequested || found.Request == nil || !found.Request.Pending() {
		t.Fatalf("after asking = %+v, %v, want requested with the pending request", found, err)
	}

	seatLeader(t, mgr, g.ID, "leader-1")
	if _, err := mgr.DeclineJoinRequest(ctx, found.Request.ID, "leader-1", true, ""); err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	found, err = mgr.FindGroupByCode(ctx, g.Code, a.ID)
	if err != nil || found.State != models.CallerDeclined || found.Request == nil || !found.Request.Silent {
		t.Fatalf("after a quiet decline = %+v, %v, want declined carrying the silent request", found, err)
	}

	if _, err := mgr.JoinGroupByCode(ctx, g.Code, a.ID, "Please."); err != nil {
		t.Fatalf("appeal by code: %v", err)
	}
	pending, _ := mgr.FindGroupByCode(ctx, g.Code, a.ID)
	if _, err := mgr.ApproveJoinRequest(ctx, pending.Request.ID, "leader-1"); err != nil {
		t.Fatalf("ApproveJoinRequest: %v", err)
	}
	found, err = mgr.FindGroupByCode(ctx, g.Code, a.ID)
	if err != nil || found.State != models.CallerIsMember || found.Request != nil {
		t.Errorf("after approval = %+v, %v, want member", found, err)
	}
}

func TestJoiningByCodeFollowsTheJoinModeOfAHiddenGroup(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := hiddenGroup(t, mgr, models.JoinOpen)

	if _, err := mgr.JoinGroup(ctx, g.ID, a.ID, ""); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Errorf("joining a hidden group by id err = %v, want ErrGroupNotFound", err)
	}
	out, err := mgr.JoinGroupByCode(ctx, g.Code, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroupByCode: %v", err)
	}
	if out.Status != models.JoinedNow || !registered(t, mgr, a.ID, g.ID) {
		t.Errorf("join by code = %+v, want joined at once", out)
	}
	again, err := mgr.JoinGroupByCode(ctx, g.Code, a.ID, "")
	if err != nil || again.Status != models.AlreadyAMember {
		t.Errorf("joining again by code = %+v, %v, want member", again, err)
	}
	joinable, err := mgr.ListJoinableGroups(ctx, "someone-else")
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	if len(joinable) != 0 {
		t.Errorf("joinable = %v, want the hidden group still unlisted", joinable)
	}
}

func TestAnUnknownCodeAndAnotherAgencysCodeAnswerAlike(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	ctx := context.Background()
	here := newTestManagerOn(t, db)
	elsewhereSpec, err := mwanachamaactor.SpecFor("elsewhere")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaactor.Provision(db, elsewhereSpec); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	elsewhere, err := mwanachamaactor.NewUserManager(db, elsewhereSpec)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	_, a := seedJoin(t, here, models.JoinOpen)
	if _, err := elsewhere.CreateGroup(ctx, mwanachamaactor.Group{Name: "First"}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	theirs, err := elsewhere.CreateGroup(ctx, mwanachamaactor.Group{Name: "Debrecen residents", JoinMode: models.JoinOpen})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	for _, code := range []string{theirs.Code, "G-999", "", "not-a-code"} {
		if _, err := here.FindGroupByCode(ctx, code, a.ID); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
			t.Errorf("find %q err = %v, want ErrGroupNotFound", code, err)
		}
		if _, err := here.JoinGroupByCode(ctx, code, a.ID, ""); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
			t.Errorf("join %q err = %v, want ErrGroupNotFound", code, err)
		}
	}
	if registered(t, elsewhere, a.ID, theirs.ID) {
		t.Error("a code from another agency registered the actor there")
	}
}

func TestJoinableGroupsCarryTheirCode(t *testing.T) {
	mgr := newTestManager(t)
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	joinable, err := mgr.ListJoinableGroups(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("ListJoinableGroups: %v", err)
	}
	if len(joinable) != 1 || joinable[0].Code != g.Code || !strings.HasPrefix(joinable[0].Code, "G-") {
		t.Errorf("joinable = %+v, want the group with its G-<n> code", joinable)
	}
}
