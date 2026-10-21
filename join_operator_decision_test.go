package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestAnOperatorWhoLeadsNothingApprovesAJoinRequest(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	res, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.ApproveJoinRequest(ctx, res.Request.ID, "owner-1"); !errors.Is(err, mwanachamaactor.ErrNotGroupLeader) {
		t.Fatalf("leader path for a non-leader = %v, want ErrNotGroupLeader", err)
	}
	got, err := mgr.OperatorApproveJoinRequest(ctx, res.Request.ID, "owner-1")
	if err != nil {
		t.Fatalf("OperatorApproveJoinRequest: %v", err)
	}
	if got.State != models.JoinRequestApproved || got.DecidedBy != "owner-1" {
		t.Errorf("request = %+v, want approved by the operator", got)
	}
	decisions, err := mgr.ListJoinDecisions(ctx, res.Request.ID)
	if err != nil || len(decisions) != 1 || decisions[0].DecidedBy != "owner-1" {
		t.Fatalf("decisions = %+v, %v; want the operator's approval kept", decisions, err)
	}
	if _, err := mgr.OperatorApproveJoinRequest(ctx, res.Request.ID, "owner-1"); !errors.Is(err, mwanachamaactor.ErrJoinRequestDecided) {
		t.Errorf("second approval = %v, want ErrJoinRequestDecided", err)
	}
}

func TestAnOperatorDeclinesQuietlyWithoutAReason(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	res, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.OperatorDeclineJoinRequest(ctx, res.Request.ID, "agent-1", true, "Not here."); !errors.Is(err, mwanachamaactor.ErrQuietDeclineReason) {
		t.Fatalf("quiet decline with a reason = %v, want ErrQuietDeclineReason", err)
	}
	got, err := mgr.OperatorDeclineJoinRequest(ctx, res.Request.ID, "agent-1", true, "")
	if err != nil {
		t.Fatalf("OperatorDeclineJoinRequest: %v", err)
	}
	if got.State != models.JoinRequestDeclined || !got.Silent || got.DeclineReason != "" || got.DecidedBy != "agent-1" {
		t.Errorf("request = %+v, want a quiet reasonless decline by the operator", got)
	}
}

func TestAnOperatorDecisionStillNamesWhoMadeIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	res, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if _, err := mgr.OperatorApproveJoinRequest(ctx, res.Request.ID, " "); !errors.Is(err, mwanachamaactor.ErrInvalidJoinRequest) {
		t.Errorf("anonymous operator approval = %v, want ErrInvalidJoinRequest", err)
	}
}
