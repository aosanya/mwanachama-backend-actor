package mwanachamaactor_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestAskedAtIsTheLastAskAndADecisionLeavesItAlone(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")

	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "Please.")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if first.Request.AskedAt == "" || first.Request.AskedAt != first.Request.RequestedAt {
		t.Fatalf("first ask = %+v, want asked_at equal to requested_at", first.Request)
	}

	time.Sleep(2 * time.Millisecond)
	declined, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", true)
	if err != nil {
		t.Fatalf("DeclineJoinRequest: %v", err)
	}
	stored, err := mgr.GetJoinRequest(ctx, first.Request.ID)
	if err != nil {
		t.Fatalf("GetJoinRequest: %v", err)
	}
	if stored.AskedAt != first.Request.AskedAt || declined.AskedAt != first.Request.AskedAt {
		t.Errorf("after a decline asked_at = %q (returned %q), want the ask's %q", stored.AskedAt, declined.AskedAt, first.Request.AskedAt)
	}
	if stored.LastUpdated == first.Request.AskedAt {
		t.Errorf("last_updated = %q, want the decline to have moved it", stored.LastUpdated)
	}

	time.Sleep(2 * time.Millisecond)
	appeal, err := mgr.JoinGroup(ctx, g.ID, a.ID, "I have moved here since.")
	if err != nil {
		t.Fatalf("JoinGroup appeal: %v", err)
	}
	r := appeal.Request
	if r.AskedAt == first.Request.AskedAt || r.AskedAt != r.LastUpdated || r.RequestedAt != first.Request.RequestedAt {
		t.Errorf("appeal = %+v, want asked_at moved to the appeal and requested_at kept", r)
	}
	reread, err := mgr.GetJoinRequest(ctx, r.ID)
	if err != nil || reread.AskedAt != r.AskedAt {
		t.Errorf("stored appeal = %+v, %v; want asked_at %q", reread, err, r.AskedAt)
	}
}

func TestProvisionAddsAskedAtToAJoinRequestTableWithoutIt(t *testing.T) {
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
	out, err := mgr.JoinGroup(context.Background(), g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	o, _ := s.ByRole("group_join_request")
	if err := db.Exec("alter table " + s.TableFor(o) + " drop column asked_at").Error; err != nil {
		t.Fatalf("drop asked_at: %v", err)
	}

	again := newTestManagerOn(t, db)
	got, err := again.GetJoinRequest(context.Background(), out.Request.ID)
	if err != nil {
		t.Fatalf("GetJoinRequest: %v", err)
	}
	if got.AskedAt != "" {
		t.Errorf("an existing request reads asked_at %q, want empty", got.AskedAt)
	}
	if _, err := again.JoinGroup(context.Background(), g.ID, a.ID, "again"); err != nil {
		t.Errorf("JoinGroup on the restored column: %v", err)
	}
}
