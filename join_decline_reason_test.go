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

func TestADeclineKeepsItsReasonForTheApplicantsOwnReads(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "Please.")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}

	declined, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", false, "  This group is for residents of Ward 4.  ")
	if err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	want := "This group is for residents of Ward 4."
	if declined.DeclineReason != want {
		t.Errorf("returned reason = %q, want %q", declined.DeclineReason, want)
	}
	stored, err := mgr.GetJoinRequest(ctx, first.Request.ID)
	if err != nil || stored.DeclineReason != want {
		t.Errorf("stored = %+v, %v; want reason %q", stored, err, want)
	}
	mine, err := mgr.ListJoinRequestsForActor(ctx, a.ID)
	if err != nil || len(mine) != 1 || mine[0].DeclineReason != want {
		t.Errorf("the applicant's list = %+v, %v; want the reason %q", mine, err, want)
	}

	appeal, err := mgr.JoinGroup(ctx, g.ID, a.ID, "I have moved here since.")
	if err != nil {
		t.Fatalf("JoinGroup appeal: %v", err)
	}
	if appeal.Request.DeclineReason != "" {
		t.Errorf("appeal reason = %q, want it cleared", appeal.Request.DeclineReason)
	}
	reread, err := mgr.GetJoinRequest(ctx, first.Request.ID)
	if err != nil || reread.DeclineReason != "" || !reread.Appeal {
		t.Errorf("stored appeal = %+v, %v; want an appeal with no reason", reread, err)
	}
}

func TestADeclineWithoutAReasonStoresNone(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	declined, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", true, "   ")
	if err != nil {
		t.Fatalf("a quiet decline with a blank reason: %v", err)
	}
	if declined.DeclineReason != "" {
		t.Errorf("reason = %q, want none", declined.DeclineReason)
	}
}

func TestAQuietDeclineRefusesAReasonAndWritesNothing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", true, "Not now."); !errors.Is(err, mwanachamaactor.ErrQuietDeclineReason) {
		t.Fatalf("a quiet decline with a reason = %v, want ErrQuietDeclineReason", err)
	}
	stored, err := mgr.GetJoinRequest(ctx, first.Request.ID)
	if err != nil || !stored.Pending() || stored.DeclineReason != "" {
		t.Errorf("after the refusal = %+v, %v; want it still pending with no reason", stored, err)
	}
}

func TestADeclineReasonIsAtMostFiveHundredCharacters(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", false, strings.Repeat("é", 501)); !errors.Is(err, mwanachamaactor.ErrDeclineReasonTooLong) {
		t.Fatalf("a 501-character reason = %v, want ErrDeclineReasonTooLong", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", false, strings.Repeat("é", 500)); err != nil {
		t.Fatalf("a 500-character reason: %v", err)
	}
}

func TestProvisionAddsDeclineReasonToAJoinRequestTableWithoutIt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	mgr := newTestManagerOn(t, db)
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	out, err := mgr.JoinGroup(context.Background(), g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	o, _ := s.ByRole("group_join_request")
	if err := db.Exec("alter table " + s.TableFor(o) + " drop column decline_reason").Error; err != nil {
		t.Fatalf("drop decline_reason: %v", err)
	}

	again := newTestManagerOn(t, db)
	got, err := again.DeclineJoinRequest(context.Background(), out.Request.ID, "leader-1", false, "Not this year.")
	if err != nil || got.DeclineReason != "Not this year." {
		t.Errorf("decline on the restored column = %+v, %v", got, err)
	}
}
