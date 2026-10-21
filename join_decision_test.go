package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestEveryDecisionOnAJoinRequestIsKeptAndNoneIsOverwritten(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "Please.")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", false, "Ward 4 only."); err != nil {
		t.Fatalf("first decline: %v", err)
	}
	if _, err := mgr.JoinGroup(ctx, g.ID, a.ID, "I moved."); err != nil {
		t.Fatalf("appeal: %v", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", true, ""); err != nil {
		t.Fatalf("quiet decline: %v", err)
	}
	if _, err := mgr.JoinGroup(ctx, g.ID, a.ID, "Once more."); err != nil {
		t.Fatalf("second appeal: %v", err)
	}
	latest, err := mgr.ApproveJoinRequest(ctx, first.Request.ID, "leader-1")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if latest.State != models.JoinRequestApproved || latest.DecidedBy != "leader-1" {
		t.Errorf("request = %+v, want it to carry the latest decision", latest)
	}

	got, err := mgr.ListJoinDecisions(ctx, first.Request.ID)
	if err != nil || len(got) != 3 {
		t.Fatalf("decisions = %+v, %v; want three", got, err)
	}
	if got[0].State != models.JoinRequestDeclined || got[0].Silent || got[0].Reason != "Ward 4 only." || got[0].DecidedBy != "leader-1" || got[0].DecidedAt == "" {
		t.Errorf("first decision = %+v, want the told decline with its reason", got[0])
	}
	if got[1].State != models.JoinRequestDeclined || !got[1].Silent || got[1].Reason != "" {
		t.Errorf("second decision = %+v, want the quiet decline", got[1])
	}
	if got[2].State != models.JoinRequestApproved || got[2].Silent || got[2].ActorID != a.ID || got[2].GroupID != g.ID {
		t.Errorf("third decision = %+v, want the approval", got[2])
	}
}

func TestTheApplicantReadsOnlyTheirOwnToldDecisions(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	seatLeader(t, mgr, g.ID, "leader-1")
	first, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", false, "Not yet."); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, err := mgr.JoinGroup(ctx, g.ID, a.ID, ""); err != nil {
		t.Fatalf("appeal: %v", err)
	}
	if _, err := mgr.DeclineJoinRequest(ctx, first.Request.ID, "leader-1", true, ""); err != nil {
		t.Fatalf("quiet decline: %v", err)
	}
	all, err := mgr.ListJoinDecisions(ctx, first.Request.ID)
	if err != nil || len(all) != 2 {
		t.Fatalf("decisions = %+v, %v", all, err)
	}
	told, quiet := all[0], all[1]

	read, err := mgr.GetMyJoinDecision(ctx, told.ID, a.ID)
	if err != nil || read.Reason != "Not yet." {
		t.Errorf("told = %+v, %v; want the decision with its reason", read, err)
	}
	for name, probe := range map[string][2]string{
		"quiet":   {quiet.ID, a.ID},
		"other":   {told.ID, "someone-else"},
		"unknown": {"no-such-decision", a.ID},
	} {
		if _, err := mgr.GetMyJoinDecision(ctx, probe[0], probe[1]); !errors.Is(err, mwanachamaactor.ErrJoinDecisionNotFound) {
			t.Errorf("%s: err = %v, want ErrJoinDecisionNotFound", name, err)
		}
	}
}
