package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func seedHidden(t *testing.T, mgr mwanachamaactor.UserManager) (mwanachamaactor.Group, mwanachamaactor.Actor) {
	t.Helper()
	g, a := seedJoin(t, mgr, "")
	if _, err := mgr.SetGroupHidden(context.Background(), g.ID, true); err != nil {
		t.Fatalf("SetGroupHidden: %v", err)
	}
	seatLeader(t, mgr, g.ID, "leader-1")
	seatLeader(t, mgr, g.ID, "leader-2")
	return g, a
}

func TestAcceptingAnInvitationRegistersAtAHiddenGroup(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)

	inv, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if inv.State != models.InvitationPending || inv.InvitedBy != "leader-1" || inv.InvitedAt == "" || inv.ExpiresAt == "" {
		t.Fatalf("invitation = %+v, want pending from leader-1 with an expiry", inv)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Fatal("an invitation alone registered the actor")
	}

	got, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID)
	if err != nil {
		t.Fatalf("AcceptInvitation: %v", err)
	}
	if got.State != models.InvitationAccepted || got.AnsweredAt == "" {
		t.Errorf("accepted = %+v, want accepted with an answer time", got)
	}
	if !registered(t, mgr, a.ID, g.ID) {
		t.Error("accepting did not register the actor")
	}
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID); !errors.Is(err, mwanachamaactor.ErrInvitationAnswered) {
		t.Errorf("accepting twice err = %v, want ErrInvitationAnswered", err)
	}
}

func TestDecliningAnInvitationRegistersNothing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)
	inv, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}

	got, err := mgr.DeclineInvitation(ctx, inv.ID, a.ID)
	if err != nil {
		t.Fatalf("DeclineInvitation: %v", err)
	}
	if got.State != models.InvitationDeclined {
		t.Errorf("declined = %+v, want declined", got)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Error("declining registered the actor")
	}
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID); !errors.Is(err, mwanachamaactor.ErrInvitationAnswered) {
		t.Errorf("accepting a declined invitation err = %v, want ErrInvitationAnswered", err)
	}

	again, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-2", "")
	if err != nil {
		t.Fatalf("InviteToGroup again: %v", err)
	}
	if again.ID != inv.ID || again.State != models.InvitationPending || again.InvitedBy != "leader-2" {
		t.Errorf("invited again = %+v, want the same invitation pending from leader-2", again)
	}
	all, err := mgr.ListInvitationsForGroup(ctx, g.ID, false)
	if err != nil || len(all) != 1 {
		t.Fatalf("ListInvitationsForGroup = %v %v, want one invitation", all, err)
	}
	if all[0].AnsweredAt != "" {
		t.Errorf("a renewed invitation kept answered_at %q", all[0].AnsweredAt)
	}
}

func TestALapsedInvitationCannotBeAnsweredAndWritesNothing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)
	expires := time.Now().Add(500 * time.Millisecond).UTC().Format(time.RFC3339Nano)
	inv, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", expires)
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	open, err := mgr.ListInvitationsForGroup(ctx, g.ID, true)
	if err != nil || len(open) != 1 {
		t.Fatalf("open invitations before lapsing = %v %v, want one", open, err)
	}
	time.Sleep(600 * time.Millisecond)

	if _, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID); !errors.Is(err, mwanachamaactor.ErrInvitationLapsed) {
		t.Errorf("accepting a lapsed invitation err = %v, want ErrInvitationLapsed", err)
	}
	if _, err := mgr.DeclineInvitation(ctx, inv.ID, a.ID); !errors.Is(err, mwanachamaactor.ErrInvitationLapsed) {
		t.Errorf("declining a lapsed invitation err = %v, want ErrInvitationLapsed", err)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Error("a lapsed invitation registered the actor")
	}
	stored, err := mgr.GetInvitation(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetInvitation: %v", err)
	}
	if stored.State != models.InvitationPending || stored.LastUpdated != inv.LastUpdated {
		t.Errorf("lapsing wrote to the invitation: %+v", stored)
	}
	open, err = mgr.ListInvitationsForGroup(ctx, g.ID, true)
	if err != nil || len(open) != 0 {
		t.Errorf("open invitations after lapsing = %v %v, want none", open, err)
	}
	mine, err := mgr.ListInvitationsForActor(ctx, a.ID)
	if err != nil || len(mine) != 1 {
		t.Errorf("ListInvitationsForActor = %v %v, want the lapsed one still listed", mine, err)
	}
}

func TestOnlyTheInvitedActorMayAnswer(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)
	inv, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, "someone-else"); !errors.Is(err, mwanachamaactor.ErrInvitationNotFound) {
		t.Errorf("another actor accepting err = %v, want ErrInvitationNotFound", err)
	}
	if _, err := mgr.DeclineInvitation(ctx, inv.ID, "someone-else"); !errors.Is(err, mwanachamaactor.ErrInvitationNotFound) {
		t.Errorf("another actor declining err = %v, want ErrInvitationNotFound", err)
	}
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, ""); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("accepting with no actor err = %v, want ErrInvalidInvitation", err)
	}
}

func TestAnInvitationIsRefusedWhereItCannotMean(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedHidden(t, mgr)

	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "", ""); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("no issuer err = %v, want ErrInvalidInvitation", err)
	}
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", "next week"); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("an unreadable expiry err = %v, want ErrInvalidInvitation", err)
	}
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", past); !errors.Is(err, mwanachamaactor.ErrInvalidInvitation) {
		t.Errorf("an expiry in the past err = %v, want ErrInvalidInvitation", err)
	}
	if _, err := mgr.InviteToGroup(ctx, "missing", a.ID, "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
		t.Errorf("a missing group err = %v, want ErrGroupNotFound", err)
	}
	if _, err := mgr.InviteToGroup(ctx, g.ID, "missing", "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Errorf("a missing actor err = %v, want ErrActorNotFound", err)
	}
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", ""); !errors.Is(err, mwanachamaactor.ErrAlreadyInGroup) {
		t.Errorf("inviting a registered actor err = %v, want ErrAlreadyInGroup", err)
	}
}

func TestAcceptingSettlesAPendingJoinRequest(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	asked, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	seatLeader(t, mgr, g.ID, "leader-1")
	inv, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID); err != nil {
		t.Fatalf("AcceptInvitation: %v", err)
	}
	req, err := mgr.GetJoinRequest(ctx, asked.Request.ID)
	if err != nil {
		t.Fatalf("GetJoinRequest: %v", err)
	}
	if req.State != models.JoinRequestApproved {
		t.Errorf("join request = %+v, want approved once the invitation registered the actor", req)
	}
}
